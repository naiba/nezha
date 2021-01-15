package main

import (
	"fmt"
	"log"
	"net"
	"os/exec"
	"time"

	"github.com/genkiroid/cert"
	"github.com/go-ping/ping"
	"github.com/shirou/gopsutil/v3/disk"
)

func main() {
	conn, err := net.DialTimeout("tcp", "example.com:80", time.Second*10)
	if err != nil {
		panic(err)
	}
	println(conn)
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
	certs, err := cert.NewCerts([]string{"example.com"})
	if err != nil {
		panic(err)
	}
	fmt.Println(certs)
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
