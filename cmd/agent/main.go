package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
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
	"google.golang.org/grpc"

	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/pkg/utils"
	pb "github.com/naiba/nezha/proto"
	"github.com/naiba/nezha/service/dao"
	"github.com/naiba/nezha/service/monitor"
	"github.com/naiba/nezha/service/rpc"
)

var (
	server       string
	clientSecret string
	version      string
)

var (
	reporting      bool
	client         pb.NezhaServiceClient
	ctx            = context.Background()
	delayWhenError = time.Second * 10       // Agent 重连间隔
	updateCh       = make(chan struct{}, 0) // Agent 自动更新间隔
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

	var debug bool
	flag.String("i", "", "unused 旧Agent兼容")
	flag.BoolVar(&debug, "d", false, "允许不安全连接")
	flag.StringVar(&server, "s", "localhost:5555", "管理面板RPC端口")
	flag.StringVar(&clientSecret, "p", "", "Agent连接Secret")
	flag.Parse()

	dao.Conf = &model.Config{
		Debug: debug,
	}

	if server == "" || clientSecret == "" {
		flag.Usage()
		return
	}

	run()
}

func run() {
	auth := rpc.AuthHandler{
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
		var cmd *exec.Cmd
		var endCh = make(chan struct{})
		pg, err := utils.NewProcessExitGroup()
		if err != nil {
			// 进程组创建失败，直接退出
			result.Data = err.Error()
			client.ReportTask(ctx, &result)
			return
		}
		timeout := time.NewTimer(time.Hour * 2)
		if utils.IsWindows() {
			cmd = exec.Command("cmd", "/c", task.GetData())
		} else {
			cmd = exec.Command("sh", "-c", task.GetData())
		}
		pg.AddProcess(cmd)
		go func() {
			select {
			case <-timeout.C:
				result.Data = "任务执行超时\n"
				close(endCh)
				pg.Dispose()
			case <-endCh:
				timeout.Stop()
			}
		}()
		output, err := cmd.Output()
		if err != nil {
			result.Data += fmt.Sprintf("%s\n%s", string(output), err.Error())
		} else {
			close(endCh)
			result.Data = string(output)
			result.Successful = true
		}
		result.Delay = float32(time.Now().Sub(startedAt).Seconds())
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
