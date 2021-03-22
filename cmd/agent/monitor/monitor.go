package monitor

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"

	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/service/dao"
)

var netInSpeed, netOutSpeed, netInTransfer, netOutTransfer, lastUpdate uint64

func GetHost() *model.Host {
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
	mv, _ := mem.VirtualMemory()
	ms, _ := mem.SwapMemory()
	u, _ := disk.Usage("/")

	return &model.Host{
		Platform:        hi.OS,
		PlatformVersion: hi.PlatformVersion,
		CPU:             cpus,
		MemTotal:        mv.Total,
		DiskTotal:       u.Total,
		SwapTotal:       ms.Total,
		Arch:            hi.KernelArch,
		Virtualization:  hi.VirtualizationSystem,
		BootTime:        hi.BootTime,
		IP:              cachedIP,
		CountryCode:     strings.ToLower(cachedCountry),
		Version:         dao.Version,
	}
}

func GetState(delay int64) *model.HostState {
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
	u, _ := disk.Usage("/")

	return &model.HostState{
		CPU:            cpuPercent,
		MemUsed:        mv.Used,
		SwapUsed:       ms.Used,
		DiskUsed:       u.Used,
		NetInTransfer:  atomic.LoadUint64(&netInTransfer),
		NetOutTransfer: atomic.LoadUint64(&netOutTransfer),
		NetInSpeed:     atomic.LoadUint64(&netInSpeed),
		NetOutSpeed:    atomic.LoadUint64(&netOutSpeed),
		Uptime:         hi.Uptime,
	}
}

func TrackNetworkSpeed() {
	var innerNetInTransfer, innerNetOutTransfer uint64
	nc, err := net.IOCounters(false)
	if err == nil {
		innerNetInTransfer += nc[0].BytesRecv
		innerNetOutTransfer += nc[0].BytesSent
		now := uint64(time.Now().Unix())
		diff := now - atomic.LoadUint64(&lastUpdate)
		if diff > 0 {
			atomic.StoreUint64(&netInSpeed, (innerNetInTransfer-atomic.LoadUint64(&netInTransfer))/diff)
			atomic.StoreUint64(&netOutSpeed, (innerNetOutTransfer-atomic.LoadUint64(&netOutTransfer))/diff)
		}
		atomic.StoreUint64(&netInTransfer, innerNetInTransfer)
		atomic.StoreUint64(&netOutTransfer, innerNetOutTransfer)
		atomic.StoreUint64(&lastUpdate, now)
	}
}
