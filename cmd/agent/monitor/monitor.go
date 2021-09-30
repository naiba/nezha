package monitor

import (
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"

	"github.com/naiba/nezha/model"
)

var (
	Version           string = "debug"
	expectDiskFsTypes        = []string{
		"apfs", "ext4", "ext3", "ext2", "f2fs", "reiserfs", "jfs", "btrfs",
		"fuseblk", "zfs", "simfs", "ntfs", "fat32", "exfat", "xfs", "fuse.rclone",
	}
	excludeNetInterfaces = []string{
		"lo", "tun", "docker", "veth", "br-", "vmbr", "vnet", "kube",
	}
	getMacDiskNo = regexp.MustCompile(`\/dev\/disk(\d)s.*`)
)

var (
	netInSpeed, netOutSpeed, netInTransfer, netOutTransfer, lastUpdateNetStats uint64
	cachedBootTime                                                             time.Time
)

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
	diskTotal, _ := getDiskTotalAndUsed()

	var swapMemTotal uint64
	if runtime.GOOS == "windows" {
		ms, _ := mem.SwapMemory()
		swapMemTotal = ms.Total
	} else {
		swapMemTotal = mv.SwapTotal
	}

	if cachedBootTime.IsZero() {
		cachedBootTime = time.Unix(int64(hi.BootTime), 0)
	}

	return &model.Host{
		Platform:        hi.OS,
		PlatformVersion: hi.PlatformVersion,
		CPU:             cpus,
		MemTotal:        mv.Total,
		SwapTotal:       swapMemTotal,
		DiskTotal:       diskTotal,
		Arch:            hi.KernelArch,
		Virtualization:  hi.VirtualizationSystem,
		BootTime:        hi.BootTime,
		IP:              cachedIP,
		CountryCode:     strings.ToLower(cachedCountry),
		Version:         Version,
	}
}

func GetState(skipConnectionCount bool, skipProcsCount bool) *model.HostState {
	var procs []int32
	if !skipProcsCount {
		procs, _ = process.Pids()
	}

	mv, _ := mem.VirtualMemory()

	var swapMemUsed uint64
	if runtime.GOOS == "windows" {
		// gopsutil 在 Windows 下不能正确取 swap
		ms, _ := mem.SwapMemory()
		swapMemUsed = ms.Used
	} else {
		swapMemUsed = mv.SwapTotal - mv.SwapFree
	}

	var cpuPercent float64
	cp, err := cpu.Percent(0, false)
	if err == nil {
		cpuPercent = cp[0]
	}

	_, diskUsed := getDiskTotalAndUsed()
	loadStat, _ := load.Avg()

	var tcpConnCount, udpConnCount uint64
	if !skipConnectionCount {
		conns, _ := net.Connections("all")
		for i := 0; i < len(conns); i++ {
			switch conns[i].Type {
			case syscall.SOCK_STREAM:
				tcpConnCount++
			case syscall.SOCK_DGRAM:
				udpConnCount++
			}
		}
	}

	return &model.HostState{
		CPU:            cpuPercent,
		MemUsed:        mv.Total - mv.Available,
		SwapUsed:       swapMemUsed,
		DiskUsed:       diskUsed,
		NetInTransfer:  netInTransfer,
		NetOutTransfer: netOutTransfer,
		NetInSpeed:     netInSpeed,
		NetOutSpeed:    netOutSpeed,
		Uptime:         uint64(time.Since(cachedBootTime).Seconds()),
		Load1:          loadStat.Load1,
		Load5:          loadStat.Load5,
		Load15:         loadStat.Load15,
		TcpConnCount:   tcpConnCount,
		UdpConnCount:   udpConnCount,
		ProcessCount:   uint64(len(procs)),
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
		diff := now - lastUpdateNetStats
		if diff > 0 {
			netInSpeed = (innerNetInTransfer - netInTransfer) / diff
			netOutSpeed = (innerNetOutTransfer - netOutTransfer) / diff
		}
		netInTransfer = innerNetInTransfer
		netOutTransfer = innerNetOutTransfer
		lastUpdateNetStats = now
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
		if strings.Contains(str, list[i]) {
			return true
		}
	}
	return false
}
