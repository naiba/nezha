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
	"net/url"
	"os"
	"os/exec"
	"time"

	"github.com/blang/semver"
	"github.com/genkiroid/cert"
	"github.com/go-ping/ping"
	"github.com/p14yground/go-github-selfupdate/selfupdate"
	"google.golang.org/grpc"

	"github.com/naiba/nezha/cmd/agent/monitor"
	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/pkg/utils"
	pb "github.com/naiba/nezha/proto"
	"github.com/naiba/nezha/service/rpc"
)

func init() {
	cert.TimeoutSeconds = 30
	http.DefaultClient.Timeout = time.Second * 5
}

var (
	server, clientSecret, version string
	debug                         bool
	stateConf                     monitor.GetStateConfig
)

var (
	client     pb.NezhaServiceClient
	inited     bool
	updateCh   = make(chan struct{}) // Agent 自动更新间隔
	httpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: time.Second * 30,
	}
)

const (
	delayWhenError = time.Second * 10 // Agent 重连间隔
	networkTimeOut = time.Second * 5  // 普通网络超时
)

func main() {
	// 来自于 GoReleaser 的版本号
	monitor.Version = version

	flag.String("i", "", "unused 旧Agent配置兼容")
	flag.BoolVar(&debug, "d", false, "开启调试信息")
	flag.StringVar(&server, "s", "localhost:5555", "管理面板RPC端口")
	flag.StringVar(&clientSecret, "p", "", "Agent连接Secret")
	flag.BoolVar(&stateConf.SkipConnectionCount, "kconn", false, "不监控连接数")
	flag.BoolVar(&stateConf.SkipProcessCount, "kprocess", false, "不监控进程数")
	flag.Parse()

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
	// 更新IP信息
	go monitor.UpdateIP()

	if _, err := semver.Parse(version); err == nil {
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
		inited = false
		println("Error to close connection ...")
		if conn != nil {
			conn.Close()
		}
		time.Sleep(delayWhenError)
		println("Try to reconnect ...")
	}

	for {
		timeOutCtx, cancel := context.WithTimeout(context.Background(), networkTimeOut)
		conn, err = grpc.DialContext(timeOutCtx, server, grpc.WithInsecure(), grpc.WithPerRPCCredentials(&auth))
		if err != nil {
			println("grpc.Dial err: ", err)
			cancel()
			retry()
			continue
		}
		cancel()
		client = pb.NewNezhaServiceClient(conn)
		// 第一步注册
		timeOutCtx, cancel = context.WithTimeout(context.Background(), networkTimeOut)
		_, err = client.ReportSystemInfo(timeOutCtx, monitor.GetHost().PB())
		if err != nil {
			println("client.ReportSystemInfo err: ", err)
			cancel()
			retry()
			continue
		}
		cancel()
		inited = true
		// 执行 Task
		tasks, err := client.RequestTask(context.Background(), monitor.GetHost().PB())
		if err != nil {
			println("client.RequestTask err: ", err)
			retry()
			continue
		}
		err = receiveTasks(tasks)
		println("receiveTasks exit to main: ", err)
		retry()
	}
}

func receiveTasks(tasks pb.NezhaService_RequestTaskClient) error {
	var err error
	defer println("receiveTasks exit", time.Now(), "=>", err)
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
			// 检查 HTTP Response 状态
			result.Delay = float32(time.Since(start).Microseconds()) / 1000.0
			if resp.StatusCode > 399 || resp.StatusCode < 200 {
				err = errors.New("\n应用错误：" + resp.Status)
			}
		}
		if err == nil {
			// 检查 SSL 证书信息
			serviceUrl, err := url.Parse(task.GetData())
			if err == nil {
				if serviceUrl.Scheme == "https" {
					c := cert.NewCert(serviceUrl.Host)
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
				result.Data = "URL解析错误：" + err.Error()
			}
		} else {
			// HTTP 请求失败
			result.Data = err.Error()
		}
	case model.TaskTypeICMPPing:
		pinger, err := ping.NewPinger(task.GetData())
		if err == nil {
			pinger.SetPrivileged(true)
			pinger.Count = 5
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
			result.Delay = float32(time.Since(start).Microseconds()) / 1000.0
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
			client.ReportTask(context.Background(), &result)
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
		result.Delay = float32(time.Since(startedAt).Seconds())
	default:
		println("Unknown action: ", task)
	}
	client.ReportTask(context.Background(), &result)
}

func reportState() {
	var lastReportHostInfo time.Time
	var err error
	defer println("reportState exit", time.Now(), "=>", err)
	for {
		// 为了更准确的记录时段流量，inited 后再上传状态信息
		if client != nil && inited {
			monitor.TrackNetworkSpeed()
			timeOutCtx, cancel := context.WithTimeout(context.Background(), networkTimeOut)
			_, err = client.ReportSystemState(timeOutCtx, monitor.GetState(stateConf).PB())
			cancel()
			if err != nil {
				println("reportState error", err)
				time.Sleep(delayWhenError)
			}
			if lastReportHostInfo.Before(time.Now().Add(-10 * time.Minute)) {
				lastReportHostInfo = time.Now()
				client.ReportSystemInfo(context.Background(), monitor.GetHost().PB())
			}
		}
		time.Sleep(time.Second)
	}
}

func doSelfUpdate() {
	defer func() {
		time.Sleep(time.Minute * 20)
		updateCh <- struct{}{}
	}()
	v := semver.MustParse(version)
	println("Check update", v)
	latest, err := selfupdate.UpdateSelf(v, "naiba/nezha")
	if err != nil {
		println("Binary update failed:", err)
		return
	}
	if latest.Version.Equals(v) {
		println("Current binary is up to date", version)
	} else {
		println("Upgrade successfully", latest.Version)
		os.Exit(1)
	}
}

func println(v ...interface{}) {
	if debug {
		log.Println(v...)
	}
}
