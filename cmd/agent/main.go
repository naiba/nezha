package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/genkiroid/cert"
	"github.com/go-ping/ping"
	"github.com/p14yground/go-github-selfupdate/selfupdate"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"github.com/naiba/nezha/model"
	pb "github.com/naiba/nezha/proto"
	"github.com/naiba/nezha/service/dao"
	"github.com/naiba/nezha/service/monitor"
	"github.com/naiba/nezha/service/rpc"
)

var (
	clientID     string
	server       string
	clientSecret string
	debug        bool
	version      string

	rootCmd = &cobra.Command{
		Use:   "nezha-agent",
		Short: "「哪吒面板」监控、备份、站点管理一站式服务",
		Long: `哪吒面板
================================
监控、备份、站点管理一站式服务
啦啦啦，啦啦啦，我是 mjj 小行家`,
		Run:     run,
		Version: version,
	}
)

var (
	reporting      bool
	client         pb.NezhaServiceClient
	ctx            = context.Background()
	delayWhenError = time.Second * 10
	updateCh       = make(chan struct{}, 0)
	httpClient     = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
)

func doSelfUpdate() {
	defer func() {
		time.Sleep(time.Minute * 20)
		updateCh <- struct{}{}
	}()
	v := semver.MustParse(version)
	log.Println("Check update", v)
	latest, err := selfupdate.UpdateSelf(v, "naiba/nezha")
	if err != nil {
		log.Println("Binary update failed:", err)
		return
	}
	if latest.Version.Equals(v) {
		// latest version is the same as current version. It means current binary is up to date.
		log.Println("Current binary is the latest version", version)
	} else {
		log.Println("Successfully updated to version", latest.Version)
		os.Exit(1)
	}
}

func init() {
	cert.TimeoutSeconds = 30
}

func main() {
	// 来自于 GoReleaser 的版本号
	dao.Version = version
	rootCmd.PersistentFlags().StringVarP(&server, "server", "s", "localhost:5555", "客户端ID")
	rootCmd.PersistentFlags().StringVarP(&clientID, "id", "i", "", "客户端ID")
	rootCmd.PersistentFlags().StringVarP(&clientSecret, "secret", "p", "", "客户端Secret")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "开启Debug")
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	dao.Conf = &model.Config{
		Debug: debug,
	}
	auth := rpc.AuthHandler{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}

	// 上报服务器信息
	go reportState()

	if version != "" {
		go func() {
			for range updateCh {
				go doSelfUpdate()
			}
		}()
		updateCh <- struct{}{}
	}

	var err error
	var conn *grpc.ClientConn

	retry := func() {
		log.Println("Error to close connection ...")
		if conn != nil {
			conn.Close()
		}
		time.Sleep(delayWhenError)
		log.Println("Try to reconnect ...")
	}

	for {
		conn, err = grpc.Dial(server, grpc.WithInsecure(), grpc.WithPerRPCCredentials(&auth))
		if err != nil {
			log.Printf("grpc.Dial err: %v", err)
			retry()
			continue
		}
		client = pb.NewNezhaServiceClient(conn)
		// 第一步注册
		_, err = client.ReportSystemInfo(ctx, monitor.GetHost().PB())
		if err != nil {
			log.Printf("client.ReportSystemInfo err: %v", err)
			retry()
			continue
		}
		// 执行 Task
		tasks, err := client.RequestTask(ctx, monitor.GetHost().PB())
		if err != nil {
			log.Printf("client.RequestTask err: %v", err)
			retry()
			continue
		}
		err = receiveTasks(tasks)
		log.Printf("receiveTasks exit to main: %v", err)
		retry()
	}
}

func receiveTasks(tasks pb.NezhaService_RequestTaskClient) error {
	var err error
	defer log.Printf("receiveTasks exit %v => %v", time.Now(), err)
	for {
		var task *pb.Task
		task, err = tasks.Recv()
		if err != nil {
			return err
		}
		go doTask(task)
	}
}

func doTask(task *pb.Task) {
	var result pb.TaskResult
	result.Id = task.GetId()
	result.Type = task.GetType()
	switch task.GetType() {
	case model.TaskTypeHTTPGET:
		start := time.Now()
		resp, err := httpClient.Get(task.GetData())
		if err == nil {
			result.Delay = float32(time.Now().Sub(start).Microseconds()) / 1000.0
			if resp.StatusCode > 399 || resp.StatusCode < 200 {
				err = errors.New("\n应用错误：" + resp.Status)
			}
		}
		if err == nil {
			if strings.HasPrefix(task.GetData(), "https://") {
				c := cert.NewCert(task.GetData()[8:])
				if c.Error != "" {
					result.Data = "SSL证书错误：" + c.Error
				} else {
					result.Data = c.Issuer + "|" + c.NotAfter
					result.Successful = true
				}
			} else {
				result.Successful = true
			}
		} else {
			result.Data = err.Error()
		}
	case model.TaskTypeICMPPing:
		pinger, err := ping.NewPinger(task.GetData())
		if err == nil {
			pinger.SetPrivileged(true)
			pinger.Count = 10
			pinger.Timeout = time.Second * 20
			err = pinger.Run() // Blocks until finished.
		}
		if err == nil {
			result.Delay = float32(pinger.Statistics().AvgRtt.Microseconds()) / 1000.0
			result.Successful = true
		} else {
			result.Data = err.Error()
		}
	case model.TaskTypeTCPPing:
		start := time.Now()
		conn, err := net.DialTimeout("tcp", task.GetData(), time.Second*10)
		if err == nil {
			conn.Write([]byte("ping\n"))
			conn.Close()
			result.Delay = float32(time.Now().Sub(start).Microseconds()) / 1000.0
			result.Successful = true
		} else {
			result.Data = err.Error()
		}
	case model.TaskTypeCommand:
		startedAt := time.Now()
		cmd := exec.Command(task.GetData())
		output, err := cmd.Output()
		result.Delay = float32(time.Now().Sub(startedAt).Seconds())
		if err != nil {
			result.Data = fmt.Sprintf("%s\n%s", string(output), err.Error())
		} else {
			result.Data = string(output)
			result.Successful = true
		}
	default:
		log.Printf("Unknown action: %v", task)
	}
	client.ReportTask(ctx, &result)
}

func reportState() {
	var lastReportHostInfo time.Time
	var err error
	defer log.Printf("reportState exit %v => %v", time.Now(), err)
	for {
		if client != nil {
			monitor.TrackNetworkSpeed()
			_, err = client.ReportSystemState(ctx, monitor.GetState(dao.ReportDelay).PB())
			if err != nil {
				log.Printf("reportState error %v", err)
				time.Sleep(delayWhenError)
			}
			if lastReportHostInfo.Before(time.Now().Add(-10 * time.Minute)) {
				lastReportHostInfo = time.Now()
				client.ReportSystemInfo(ctx, monitor.GetHost().PB())
			}
		}
	}
}
