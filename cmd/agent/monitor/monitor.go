package monitor

import (
	"fmt"
	"regexp"
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
var expectDiskFsTypes = []string{
	"apfs", "ext4", "ext3", "ext2", "f2fs", "reiserfs", "jfs", "btrfs", "fuseblk", "zfs", "simfs", "ntfs", "fat32", "exfat", "xfs",
}
var excludeNetInterfaces = []string{
	"lo", "tun", "docker", "veth", "br-", "vmbr", "vnet", "kube",
}
var getMacDiskNo = regexp.MustCompile(`\/dev\/disk(\d)s.*`)

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
			if isListContainsStr(excludeNetInterfaces, v.Name) {
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
	devices := make(map[string]string)
	countedDiskForMac := make(map[string]struct{})
	for _, d := range diskList {
		fsType := strings.ToLower(d.Fstype)
		// 不统计 K8s 的虚拟挂载点：https://github.com/shirou/gopsutil/issues/1007
		if devices[d.Device] == "" && isListContainsStr(expectDiskFsTypes, fsType) && !strings.Contains(d.Mountpoint, "/var/lib/kubelet") {
			devices[d.Device] = d.Mountpoint
		}
	}
	for device, mountPath := range devices {
		diskUsageOf, _ := disk.Usage(mountPath)
		// 这里是针对 Mac 机器的处理，https://github.com/giampaolo/psutil/issues/1509
		matches := getMacDiskNo.FindStringSubmatch(device)
		if len(matches) == 2 {
			if _, has := countedDiskForMac[matches[1]]; !has {
				countedDiskForMac[matches[1]] = struct{}{}
				total += diskUsageOf.Total
			}
		} else {
			total += diskUsageOf.Total
		}
		used += diskUsageOf.Used
	}
	return
}

func isListContainsStr(list []string, str string) bool {
	for i := 0; i < len(list); i++ {
		if strings.Contains(list[i], str) {
			return true
		}
	}
	return false
}
