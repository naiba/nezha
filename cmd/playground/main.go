package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"
	"time"

	"github.com/go-ping/ping"
	"github.com/shirou/gopsutil/v3/disk"
)

func main() {
	// icmp()
	// tcpping()
	httpWithSSLInfo()
	// diskinfo()
}

func tcpping() {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", "example.com:80", time.Second*10)
	if err != nil {
		panic(err)
	}
	conn.Write([]byte("ping\n"))
	conn.Close()
	fmt.Println(time.Now().Sub(start).Microseconds(), float32(time.Now().Sub(start).Microseconds())/1000.0)
}

func diskinfo() {
	// 硬盘信息
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
	resp, err := httpClient.Get("http://mail.nai.ba")
	fmt.Println(err, resp.StatusCode)
	// SSL 证书信息获取
	// c := cert.NewCert("expired-ecc-dv.ssl.com")
	// fmt.Println(c.Error)
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
	cmd := exec.Command("ping", "example.com", "-c2")
	output, err := cmd.Output()
	log.Println("output:", string(output))
	log.Println("err:", err)

	cmd = exec.Command("ping", "example", "-c2")
	output, err = cmd.Output()
	log.Println("output:", string(output))
	log.Println("err:", err)
}
