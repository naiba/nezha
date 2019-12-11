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
	"github.com/p14yground/nezha/service/dao"
)

type ipDotSbGeoIP struct {
	CountryCode string `json:"country_code,omitempty"`
	IP          string `json:"ip,omitempty"`
}

var netInSpeed, netOutSpeed, netInTransfer, netOutTransfer, lastUpdate uint64

// GetHost ..
func GetHost() *model.Host {
	hi, _ := host.Info()
	var cpus []string
	ci, _ := cpu.Info()
	for i := 0; i < len(ci); i++ {
		cpus = append(cpus, fmt.Sprintf("%v-%vC%vT", ci[i].ModelName, ci[i].Cores, ci[i].Stepping))
	}
	mv, _ := mem.VirtualMemory()
	ms, _ := mem.SwapMemory()
	var diskTotal uint64
	dparts, _ := disk.Partitions(true)
	for _, part := range dparts {
		u, _ := disk.Usage(part.Mountpoint)
		diskTotal += u.Total
	}
	var ip ipDotSbGeoIP
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
		MemTotal:        mv.Total,
		DiskTotal:       diskTotal,
		SwapTotal:       ms.Total,
		Arch:            hi.KernelArch,
		Virtualization:  hi.VirtualizationSystem,
		BootTime:        hi.BootTime,
		IP:              ip.IP,
		CountryCode:     strings.ToLower(ip.CountryCode),
		Version:         dao.Version,
	}
}

// GetState ..
func GetState(delay int64) *model.State {
	hi, _ := host.Info()
	// Memory
	mv, _ := mem.VirtualMemory()
	ms, _ := mem.SwapMemory()
	// CPU
	var cpuPercent float64
	cp, err := cpu.Percent(time.Second*time.Duration(delay), false)
	if err == nil {
		cpuPercent = cp[0]
	}
	// Disk
	var diskUsed uint64
	dparts, _ := disk.Partitions(true)
	for _, part := range dparts {
		u, _ := disk.Usage(part.Mountpoint)
		diskUsed += u.Used
	}

	return &model.State{
		CPU:            cpuPercent,
		MemUsed:        mv.Used,
		SwapUsed:       ms.Used,
		DiskUsed:       diskUsed,
		NetInTransfer:  netInTransfer,
		NetOutTransfer: netOutTransfer,
		NetInSpeed:     netInSpeed,
		NetOutSpeed:    netOutSpeed,
		Uptime:         hi.Uptime,
	}
}

// TrackNetworkSpeed ..
func TrackNetworkSpeed() {
	var innerNetInTransfer, innerNetOutTransfer uint64
	nc, err := net.IOCounters(false)
	if err == nil {
		innerNetInTransfer += nc[0].BytesRecv
		innerNetOutTransfer += nc[0].BytesSent
		if netInTransfer == 0 {
			netInTransfer = innerNetInTransfer
		}
		if netOutTransfer == 0 {
			netOutTransfer = innerNetOutTransfer
		}
		diff := uint64(time.Now().Unix())
		if lastUpdate == 0 {
			lastUpdate = diff
			return
		}
		diff -= lastUpdate
		if diff > 0 {
			netInSpeed = (innerNetInTransfer - netInTransfer) / diff
			netOutSpeed = (innerNetOutTransfer - netOutTransfer) / diff
		}
	}
}
