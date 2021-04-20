package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/genkiroid/cert"
	"github.com/go-ping/ping"
	"github.com/naiba/nezha/pkg/utils"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
)

func main() {
	// icmp()
	// tcpping()
	httpWithSSLInfo()
	// sysinfo()
	// cmdExec()
}

func tcpping() {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", "example.com:80", time.Second*10)
	if err != nil {
		panic(err)
	}
	conn.Write([]byte("ping\n"))
	conn.Close()
	fmt.Println(time.Since(start).Microseconds(), float32(time.Since(start).Microseconds())/1000.0)
}

func sysinfo() {
	hi, _ := host.Info()
	var cpuType string
	if hi.VirtualizationSystem != "" {
		cpuType = "Virtual"
	} else {
		cpuType = "Physical"
	}
	cpuModelCount := make(map[string]int)
	ci, _ := cpu.Info()
	for i := 0; i < len(ci); i++ {
		cpuModelCount[ci[i].ModelName]++
	}
	var cpus []string
	for model, count := range cpuModelCount {
		cpus = append(cpus, fmt.Sprintf("%s %d %s Core", model, count, cpuType))
	}
	os.Exit(0)
	// 硬盘信息，不使用的原因是会重复统计 Linux、Mac
	dparts, _ := disk.Partitions(false)
	for _, part := range dparts {
		u, _ := disk.Usage(part.Mountpoint)
		if u != nil {
			log.Printf("%s %d %d", part.Device, u.Total, u.Used)
		}
	}
}

func httpWithSSLInfo() {
	// 跳过 SSL 检查
	transCfg := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient := &http.Client{Transport: transCfg, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	url := "https://ops.naibahq.com"
	resp, err := httpClient.Get(url)
	fmt.Println(err, resp)
	// SSL 证书信息获取
	c := cert.NewCert(url[8:])
	fmt.Println(c.Error)
}

func icmp() {
	pinger, err := ping.NewPinger("10.10.10.2")
	if err != nil {
		panic(err) // Blocks until finished.
	}
	pinger.Count = 3000
	pinger.Timeout = 10 * time.Second
	if err = pinger.Run(); err != nil {
		panic(err)
	}
	fmt.Println(pinger.PacketsRecv, float32(pinger.Statistics().AvgRtt.Microseconds())/1000.0)
}

func cmdExec() {
	execFrom, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	var cmd *exec.Cmd
	pg, err := utils.NewProcessExitGroup()
	if err != nil {
		panic(err)
	}
	if utils.IsWindows() {
		cmd = exec.Command("cmd", "/c", os.Args[1])
		// cmd = exec.Command("cmd", "/c", execFrom+"/cmd/playground/example.sh hello asd")
	} else {
		cmd = exec.Command("sh", "-c", execFrom+`/cmd/playground/example.sh hello && \
			echo world!`)
	}
	pg.AddProcess(cmd)
	go func() {
		time.Sleep(time.Second * 10)
		if err = pg.Dispose(); err != nil {
			panic(err)
		}
		fmt.Println("killed")
	}()
	output, err := cmd.Output()
	log.Println("output:", string(output))
	log.Println("err:", err)
}
