package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/naiba/nezha/cmd/dashboard/controller"
	"github.com/naiba/nezha/cmd/dashboard/rpc"
	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/service/singleton"
	"github.com/ory/graceful"
	flag "github.com/spf13/pflag"
)

type DashboardCliParam struct {
	Version          bool   // 当前版本号
	ConfigFile       string // 配置文件路径
	DatebaseLocation string // Sqlite3 数据库文件路径
}

var (
	dashboardCliParam DashboardCliParam
)

func init() {
	flag.CommandLine.ParseErrorsWhitelist.UnknownFlags = true
	flag.BoolVarP(&dashboardCliParam.Version, "version", "v", false, "查看当前版本号")
	flag.StringVarP(&dashboardCliParam.ConfigFile, "config", "c", "data/config.yaml", "配置文件路径")
	flag.StringVar(&dashboardCliParam.DatebaseLocation, "db", "data/sqlite.db", "Sqlite3数据库文件路径")
	flag.Parse()

	// 初始化 dao 包
	singleton.InitConfigFromPath(dashboardCliParam.ConfigFile)
	singleton.InitTimezoneAndCache()
	singleton.InitDBFromPath(dashboardCliParam.DatebaseLocation)
	singleton.InitLocalizer()
	initSystem()
}

func secondsToCronString(seconds uint32) (string, error) {
	if seconds > 86400 {
		return "", errors.New("时间不能超过24小时")
	}

	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secondsRemainder := seconds % 60 // 保留剩余的秒数

	cronExpr := fmt.Sprintf("%d %d %d * * *", secondsRemainder, minutes, hours)

	return cronExpr, nil
}

func initSystem() {
	// 启动 singleton 包下的所有服务
	singleton.LoadSingleton()

	// 每天的3:30 对 监控记录 和 流量记录 进行清理
	if _, err := singleton.Cron.AddFunc("0 30 3 * * *", singleton.CleanMonitorHistory); err != nil {
		panic(err)
	}

	// 每小时对流量记录进行打点
	if _, err := singleton.Cron.AddFunc("0 0 * * * *", singleton.RecordTransferHourlyUsage); err != nil {
		panic(err)
	}

	// 按用户设置的时间间隔更新DDNS信息
	if singleton.Conf.EnableDDNS {
		if singleton.Conf.DDNSCheckPeriod == 0 {
			log.Printf("NEZHA>> DDNSCheckPeriod设置为0时不会启用DDNS")
		}
		if singleton.Conf.DDNSBaseDomain == "" {
			panic(errors.New("启用DDNS时DDNSBaseDomain不能为空"))
		}
		ddnsCronString, err := secondsToCronString(singleton.Conf.DDNSCheckPeriod)
		if err != nil {
			panic(err)
		}
		if _, err := singleton.Cron.AddFunc(ddnsCronString, singleton.RefreshDDNSRecords); err != nil {
			panic(err)
		}
	}
}

func main() {
	if dashboardCliParam.Version {
		fmt.Println(singleton.Version)
		return
	}

	singleton.CleanMonitorHistory()
	go rpc.ServeRPC(singleton.Conf.GRPCPort)
	serviceSentinelDispatchBus := make(chan model.Monitor) // 用于传递服务监控任务信息的channel
	go rpc.DispatchTask(serviceSentinelDispatchBus)
	go rpc.DispatchKeepalive()
	go singleton.AlertSentinelStart()
	singleton.NewServiceSentinel(serviceSentinelDispatchBus)
	srv := controller.ServeWeb(singleton.Conf.HTTPPort)
	if err := graceful.Graceful(func() error {
		return srv.ListenAndServe()
	}, func(c context.Context) error {
		log.Println("NEZHA>> Graceful::START")
		singleton.RecordTransferHourlyUsage()
		log.Println("NEZHA>> Graceful::END")
		srv.Shutdown(c)
		return nil
	}); err != nil {
		log.Printf("NEZHA>> ERROR: %v", err)
	}
}
