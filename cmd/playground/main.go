package main

import (
	"log"
	"os/exec"

	"github.com/shirou/gopsutil/disk"
)

func main() {
	dparts, _ := disk.Partitions(false)
	for _, part := range dparts {
		u, _ := disk.Usage(part.Mountpoint)
		log.Printf("Part:%v", part)
		log.Printf("Usage:%v", u)
	}
}

func cmdExec() {
	cmd := exec.Command("ping", "qiongbi.net", "-c2")
	output, err := cmd.Output()
	log.Println("output:", string(output))
	log.Println("err:", err)

	cmd = exec.Command("ping", "qiongbi", "-c2")
	output, err = cmd.Output()
	log.Println("output:", string(output))
	log.Println("err:", err)
}
