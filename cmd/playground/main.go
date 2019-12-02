package main

import (
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
)

func main() {
	// Host info
	hi, _ := host.Info()
	fmt.Printf("「HostInfo」 platform:%v platformVersion:%v kernelArch:%v virtualizationSystem:%v uptime:%v boottime:%v\n", hi.OS, hi.PlatformVersion, hi.KernelArch, hi.VirtualizationSystem, hi.Uptime, hi.BootTime)
	// Memory
	mv, _ := mem.VirtualMemory()
	ms, _ := mem.SwapMemory()
	fmt.Printf("「VirtualMemory」 Total: %v, Free:%v, UsedPercent:%f%%\n", mv.Total, mv.Free, mv.UsedPercent)
	fmt.Printf("「SwapMemory」 Total: %v, Free:%v, UsedPercent:%f%%\n", ms.Total, ms.Free, ms.UsedPercent)
	// Disk
	dparts, _ := disk.Partitions(false)
	for _, part := range dparts {
		fmt.Printf("「Disk」 %v\n", part)
		u, _ := disk.Usage(part.Mountpoint)
		fmt.Println("\t" + u.Path + "\t" + strconv.FormatFloat(u.UsedPercent, 'f', 2, 64) + "% full.")
		fmt.Println("\t\tTotal: " + strconv.FormatUint(u.Total/1024/1024/1024, 10) + " GiB")
		fmt.Println("\t\tFree:  " + strconv.FormatUint(u.Free/1024/1024/1024, 10) + " GiB")
		fmt.Println("\t\tUsed:  " + strconv.FormatUint(u.Used/1024/1024/1024, 10) + " GiB")
	}
	// CPU
	go func() {
		cp, _ := cpu.Percent(time.Second*2, false)
		ci, _ := cpu.Info()
		for i := 0; i < len(ci); i++ {
			fmt.Printf("「CPU」 %v core:%v step:%v", ci[i].ModelName, ci[i].Cores, ci[i].Stepping)
		}
		fmt.Printf(" percentIn2sec:%v%%\n", cp[0])
	}()
	// Network
	nc, _ := net.IOCounters(true)
	for _, ni := range nc {
		fmt.Printf("「Net」%v\n", ni)
	}
	select {}
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
