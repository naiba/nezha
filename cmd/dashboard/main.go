package main

import (
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/robfig/cron/v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/naiba/nezha/cmd/dashboard/controller"
	"github.com/naiba/nezha/cmd/dashboard/rpc"
	"github.com/naiba/nezha/model"
	pb "github.com/naiba/nezha/proto"
	"github.com/naiba/nezha/service/dao"
)

func init() {
	shanghai, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		panic(err)
	}

	// 初始化 dao 包
	dao.Conf = &model.Config{}
	dao.Cron = cron.New(cron.WithLocation(shanghai))
	dao.Crons = make(map[uint64]*model.Cron)
	dao.ServerList = make(map[uint64]*model.Server)
	dao.SecretToID = make(map[string]uint64)

	err = dao.Conf.Read("data/config.yaml")
	if err != nil {
		panic(err)
	}
	dao.DB, err = gorm.Open(sqlite.Open("data/sqlite.db"), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	if dao.Conf.Debug {
		dao.DB = dao.DB.Debug()
	}
	if dao.Conf.GRPCPort == 0 {
		dao.Conf.GRPCPort = 5555
	}
	dao.Cache = cache.New(5*time.Minute, 10*time.Minute)

	initSystem()
}

func initSystem() {
	dao.DB.AutoMigrate(model.Server{}, model.User{},
		model.Notification{}, model.AlertRule{}, model.Monitor{},
		model.MonitorHistory{}, model.Cron{})
	dao.NewServiceSentinel()

	loadServers() //加载服务器列表
	loadCrons()   //加载计划任务

	// 清理旧数据
	dao.Cron.AddFunc("* 3 * * *", cleanMonitorHistory)
}

func cleanMonitorHistory() {
	dao.DB.Delete(&model.MonitorHistory{}, "created_at < ?", time.Now().AddDate(0, 0, -30))
}

func loadServers() {
	var servers []model.Server
	dao.DB.Find(&servers)
	for _, s := range servers {
		innerS := s
		innerS.Host = &model.Host{}
		innerS.State = &model.HostState{}
		dao.ServerList[innerS.ID] = &innerS
		dao.SecretToID[innerS.Secret] = innerS.ID
	}
	dao.ReSortServer()
}

func loadCrons() {
	var crons []model.Cron
	dao.DB.Find(&crons)
	var err error
	for i := 0; i < len(crons); i++ {
		cr := crons[i]
		cr.CronID, err = dao.Cron.AddFunc(cr.Scheduler, func() {
			dao.ServerLock.RLock()
			defer dao.ServerLock.RUnlock()
			for j := 0; j < len(cr.Servers); j++ {
				if dao.ServerList[cr.Servers[j]].TaskStream != nil {
					dao.ServerList[cr.Servers[j]].TaskStream.Send(&pb.Task{
						Id:   cr.ID,
						Data: cr.Command,
						Type: model.TaskTypeCommand,
					})
				} else {
					dao.SendNotification(fmt.Sprintf("计划任务：%s，服务器：%s 离线，无法执行。", cr.Name, dao.ServerList[cr.Servers[j]].Name), false)
				}
			}
		})
		if err != nil {
			panic(err)
		}
		dao.Crons[cr.ID] = &cr
	}
	dao.Cron.Start()
}

func main() {
	go controller.ServeWeb(dao.Conf.HTTPPort)
	go rpc.ServeRPC(dao.Conf.GRPCPort)
	go rpc.DispatchTask(time.Second * 30)
	dao.AlertSentinelStart()
}
