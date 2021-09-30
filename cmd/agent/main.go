package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/blang/semver"
	"github.com/go-ping/ping"
	"github.com/gorilla/websocket"
	"github.com/p14yground/go-github-selfupdate/selfupdate"
	flag "github.com/spf13/pflag"
	"google.golang.org/grpc"

	"github.com/naiba/nezha/cmd/agent/monitor"
	"github.com/naiba/nezha/cmd/agent/processgroup"
	"github.com/naiba/nezha/cmd/agent/pty"
	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/pkg/utils"
	pb "github.com/naiba/nezha/proto"
	"github.com/naiba/nezha/service/rpc"
)

type AgentConfig struct {
	SkipConnectionCount bool
	SkipProcsCount      bool
	DisableAutoUpdate   bool
	Debug               bool
	Server              string
	ClientSecret        string
}

var (
	version string
	client  pb.NezhaServiceClient
	inited  bool
)

var (
	agentConf  AgentConfig
	updateCh   = make(chan struct{}) // Agent 自动更新间隔
	httpClient = &http.Client{
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

func init() {
	http.DefaultClient.Timeout = time.Second * 5
	flag.CommandLine.ParseErrorsWhitelist.UnknownFlags = true
}

func main() {
	// 来自于 GoReleaser 的版本号
	monitor.Version = version

	flag.BoolVarP(&agentConf.Debug, "debug", "d", false, "开启调试信息")
	flag.StringVarP(&agentConf.Server, "server", "s", "localhost:5555", "管理面板RPC端口")
	flag.StringVarP(&agentConf.ClientSecret, "password", "p", "", "Agent连接Secret")
	flag.BoolVar(&agentConf.SkipConnectionCount, "skip-conn", false, "不监控连接数")
	flag.BoolVar(&agentConf.SkipProcsCount, "skip-procs", false, "不监控进程数")
	flag.BoolVar(&agentConf.DisableAutoUpdate, "disable-auto-update", false, "禁用自动升级")
	flag.Parse()

	if agentConf.ClientSecret == "" {
		flag.Usage()
		return
	}

	run()
}

func run() {
	auth := rpc.AuthHandler{
		ClientSecret: agentConf.ClientSecret,
	}

	go pty.DownloadDependency()
	// 上报服务器信息
	go reportState()
	// 更新IP信息
	go monitor.UpdateIP()

	if _, err := semver.Parse(version); err == nil && !agentConf.DisableAutoUpdate {
		go func() {
			for range updateCh {
				go func() {
					defer func() {
						time.Sleep(time.Minute * 20)
						updateCh <- struct{}{}
					}()
					doSelfUpdate()
				}()
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
		conn, err = grpc.DialContext(timeOutCtx, agentConf.Server, grpc.WithInsecure(), grpc.WithPerRPCCredentials(&auth))
		if err != nil {
			println("与面板建立连接失败：", err)
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
			println("上报系统信息失败：", err)
			cancel()
			retry()
			continue
		}
		cancel()
		inited = true
		// 执行 Task
		tasks, err := client.RequestTask(context.Background(), monitor.GetHost().PB())
		if err != nil {
			println("请求任务失败：", err)
			retry()
			continue
		}
		err = receiveTasks(tasks)
		println("receiveTasks exit to main：", err)
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
		go func() {
			defer func() {
				if err := recover(); err != nil {
					println("task panic", task, err)
				}
			}()
			doTask(task)
		}()
	}
}

func doTask(task *pb.Task) {
	var result pb.TaskResult
	result.Id = task.GetId()
	result.Type = task.GetType()
	switch task.GetType() {
	case model.TaskTypeTerminal:
		handleTerminalTask(task)
	case model.TaskTypeHTTPGET:
		handleHttpGetTask(task, &result)
	case model.TaskTypeICMPPing:
		handleIcmpPingTask(task, &result)
	case model.TaskTypeTCPPing:
		handleTcpPingTask(task, &result)
	case model.TaskTypeCommand:
		handleCommandTask(task, &result)
	case model.TaskTypeUpgrade:
		handleUpgradeTask(task, &result)
	default:
		println("不支持的任务：", task)
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
			_, err = client.ReportSystemState(timeOutCtx, monitor.GetState(agentConf.SkipConnectionCount, agentConf.SkipProcsCount).PB())
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
	v := semver.MustParse(version)
	println("检查更新：", v)
	latest, err := selfupdate.UpdateSelf(v, "naiba/nezha")
	if err != nil {
		println("更新失败：", err)
		return
	}
	if !latest.Version.Equals(v) {
		os.Exit(1)
	}
}

func handleUpgradeTask(task *pb.Task, result *pb.TaskResult) {
	doSelfUpdate()
}

func handleTcpPingTask(task *pb.Task, result *pb.TaskResult) {
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
}

func handleIcmpPingTask(task *pb.Task, result *pb.TaskResult) {
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
}

func handleHttpGetTask(task *pb.Task, result *pb.TaskResult) {
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
		if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
			c := resp.TLS.PeerCertificates[0]
			result.Data = c.Issuer.CommonName + "|" + c.NotAfter.In(time.Local).String()
		}
		result.Successful = true
	} else {
		// HTTP 请求失败
		result.Data = err.Error()
	}
}

func handleCommandTask(task *pb.Task, result *pb.TaskResult) {
	startedAt := time.Now()
	var cmd *exec.Cmd
	var endCh = make(chan struct{})
	pg, err := processgroup.NewProcessExitGroup()
	if err != nil {
		// 进程组创建失败，直接退出
		result.Data = err.Error()
		return
	}
	timeout := time.NewTimer(time.Hour * 2)
	if utils.IsWindows() {
		cmd = exec.Command("cmd", "/c", task.GetData()) // #nosec
	} else {
		cmd = exec.Command("sh", "-c", task.GetData()) // #nosec
	}
	cmd.Env = os.Environ()
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
	pg.Dispose()
	result.Delay = float32(time.Since(startedAt).Seconds())
}

type WindowSize struct {
	Cols uint32
	Rows uint32
}

func handleTerminalTask(task *pb.Task) {
	var terminal model.TerminalTask
	err := json.Unmarshal([]byte(task.GetData()), &terminal)
	if err != nil {
		println("Terminal 任务解析错误：", err)
		return
	}
	protocol := "ws"
	if terminal.UseSSL {
		protocol += "s"
	}
	header := http.Header{}
	header.Add("Secret", agentConf.ClientSecret)
	conn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("%s://%s/terminal/%s", protocol, terminal.Host, terminal.Session), header)
	if err != nil {
		println("Terminal 连接失败：", err)
		return
	}
	defer conn.Close()

	tty, err := pty.Start()
	if err != nil {
		println("Terminal pty.Start失败：", err)
		return
	}

	defer func() {
		err := tty.Close()
		conn.Close()
		println("terminal exit", terminal.Session, err)
	}()
	println("terminal init", terminal.Session)

	go func() {
		for {
			buf := make([]byte, 1024)
			read, err := tty.Read(buf)
			if err != nil {
				conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
				return
			}
			conn.WriteMessage(websocket.BinaryMessage, buf[:read])
		}
	}()

	for {
		messageType, reader, err := conn.NextReader()
		if err != nil {
			return
		}

		if messageType == websocket.TextMessage {
			continue
		}

		dataTypeBuf := make([]byte, 1)
		read, err := reader.Read(dataTypeBuf)
		if err != nil {
			conn.WriteMessage(websocket.TextMessage, []byte("Unable to read message type from reader"))
			return
		}

		if read != 1 {
			return
		}

		switch dataTypeBuf[0] {
		case 0:
			io.Copy(tty, reader)
		case 1:
			decoder := json.NewDecoder(reader)
			var resizeMessage WindowSize
			err := decoder.Decode(&resizeMessage)
			if err != nil {
				continue
			}
			tty.Setsize(resizeMessage.Cols, resizeMessage.Rows)
		}
	}
}

func println(v ...interface{}) {
	if agentConf.Debug {
		fmt.Printf("NEZHA@%s>> ", time.Now().Format("2006-01-02 15:04:05"))
		fmt.Println(v...)
	}
}
