package monitor

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"

	"github.com/p14yground/nezha/model"
)

type ipDotSbGeoIP struct {
	CountryCode string
	IP          string
}

// GetHost ..
func GetHost() *model.Host {
	hi, _ := host.Info()
	var cpus []string
	ci, _ := cpu.Info()
	for i := 0; i < len(ci); i++ {
		cpus = append(cpus, fmt.Sprintf("%v-%vC%vT", ci[i].ModelName, ci[i].Cores, ci[i].Stepping))
	}
	ip := ipDotSbGeoIP{
		IP:          "127.0.0.1",
		CountryCode: "cn",
	}
	resp, err := http.Get("https://api.ip.sb/geoip")
	if err == nil {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		json.Unmarshal(body, &ip)
	}
	return &model.Host{
		Platform:        hi.OS,
		PlatformVersion: hi.PlatformVersion,
		CPU:             cpus,
		Arch:            hi.KernelArch,
		Virtualization:  hi.VirtualizationSystem,
		Uptime:          fmt.Sprintf("%v", hi.Uptime),
		BootTime:        fmt.Sprintf("%v", hi.BootTime),
		IP:              ip.IP,
		CountryCode:     strings.ToLower(ip.CountryCode),
	}
}

// GetState ..
func GetState(delay uint64) *model.State {
	// Memory
	mv, _ := mem.VirtualMemory()
	ms, _ := mem.SwapMemory()
	// Disk
	var diskTotal, diskUsed uint64
	dparts, _ := disk.Partitions(true)
	for _, part := range dparts {
		u, _ := disk.Usage(part.Mountpoint)
		diskTotal += u.Total
		diskUsed += u.Used
	}
	// CPU
	var cpuPercent float64
	cp, err := cpu.Percent(time.Second*time.Duration(delay), false)
	if err == nil {
		cpuPercent = cp[0]

	}
	// Network
	var netIn, netOut uint64
	nc, err := net.IOCounters(false)
	if err == nil {
		netIn = nc[0].BytesRecv
		netOut = nc[0].BytesSent
	}
	return &model.State{
		CPU:       cpuPercent,
		MEMTotal:  mv.Total,
		MEMUsed:   mv.Used,
		SwapTotal: ms.Total,
		SwapUsed:  ms.Used,
		DiskTotal: diskTotal,
		DiskUsed:  diskUsed,
		NetIn:     netIn,
		NetOut:    netOut,
	}
}
