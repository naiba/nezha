package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"
	"time"

	"github.com/genkiroid/cert"
	"github.com/go-ping/ping"
	"github.com/shirou/gopsutil/v3/disk"
)

func main() {
	// 跳过 SSL 检查
	transCfg := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient := &http.Client{Transport: transCfg}
	_, err := httpClient.Get("https://expired-ecc-dv.ssl.com")
	fmt.Println(err)
	// SSL 证书信息获取
	c := cert.NewCert("expired-ecc-dv.ssl.com")
	fmt.Println(c.Error)
	// TCP
	conn, err := net.DialTimeout("tcp", "example.com:80", time.Second*10)
	if err != nil {
		panic(err)
	}
	println(conn)
	// ICMP Ping
	pinger, err := ping.NewPinger("example.com")
	if err != nil {
		panic(err)
	}
	pinger.Count = 3
	err = pinger.Run() // Blocks until finished.
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v", pinger.Statistics())
	// 硬盘信息
	dparts, _ := disk.Partitions(false)
	for _, part := range dparts {
		u, _ := disk.Usage(part.Mountpoint)
		if u != nil {
			log.Printf("%s %d %d", part.Device, u.Total, u.Used)
		}
	}
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
