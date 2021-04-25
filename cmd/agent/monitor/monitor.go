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
		cpuType = "Vrtual"
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
	diskTotal, _ := getDiskTotalAndUsed()

	return &model.Host{
		Platform:        hi.OS,
		PlatformVersion: hi.PlatformVersion,
		CPU:             cpus,
		MemTotal:        mv.Total,
		SwapTotal:       ms.Total,
		DiskTotal:       diskTotal,
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
	mv, _ := mem.VirtualMemory()
	ms, _ := mem.SwapMemory()
	var cpuPercent float64
	cp, err := cpu.Percent(time.Second*time.Duration(delay), false)
	if err == nil {
		cpuPercent = cp[0]
	}
	_, diskUsed := getDiskTotalAndUsed()
	return &model.HostState{
		CPU:            cpuPercent,
		MemUsed:        mv.Used,
		SwapUsed:       ms.Used,
		DiskUsed:       diskUsed,
		NetInTransfer:  atomic.LoadUint64(&netInTransfer),
		NetOutTransfer: atomic.LoadUint64(&netOutTransfer),
		NetInSpeed:     atomic.LoadUint64(&netInSpeed),
		NetOutSpeed:    atomic.LoadUint64(&netOutSpeed),
		Uptime:         hi.Uptime,
	}
}

func TrackNetworkSpeed() {
	var innerNetInTransfer, innerNetOutTransfer uint64
	nc, err := net.IOCounters(true)
	if err == nil {
		for _, v := range nc {
			if strings.Contains(v.Name, "lo") ||
				strings.Contains(v.Name, "tun") ||
				strings.Contains(v.Name, "docker") ||
				strings.Contains(v.Name, "veth") ||
				strings.Contains(v.Name, "br-") ||
				strings.Contains(v.Name, "vmbr") ||
				strings.Contains(v.Name, "vnet") ||
				strings.Contains(v.Name, "kube") {
				continue
			}
			innerNetInTransfer += v.BytesRecv
			innerNetOutTransfer += v.BytesSent
		}
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

func getDiskTotalAndUsed() (total uint64, used uint64) {
	diskList, _ := disk.Partitions(false)
	for _, d := range diskList {
		fsType := strings.ToLower(d.Fstype)
		if strings.Contains(fsType, "ext4") ||
			strings.Contains(fsType, "ext3") ||
			strings.Contains(fsType, "ext2") ||
			strings.Contains(fsType, "reiserfs") ||
			strings.Contains(fsType, "jfs") ||
			strings.Contains(fsType, "btrfs") ||
			strings.Contains(fsType, "fuseblk") ||
			strings.Contains(fsType, "zfs") ||
			strings.Contains(fsType, "simfs") ||
			strings.Contains(fsType, "ntfs") ||
			strings.Contains(fsType, "fat32") ||
			strings.Contains(fsType, "exfat") ||
			strings.Contains(fsType, "xfs") {
			diskUsageOf, _ := disk.Usage(d.Mountpoint)
			path := diskUsageOf.Path
			// 不统计 K8s 的虚拟挂载点，see here：https://github.com/shirou/gopsutil/issues/1007
			if !strings.Contains(path, "/var/lib/kubelet") {
				total += diskUsageOf.Total
				used += diskUsageOf.Used
			}
		}
	}
	return
}
