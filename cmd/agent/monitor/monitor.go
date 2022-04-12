package monitor

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Erope/goss"
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
	Version           string
	expectDiskFsTypes = []string{
		"apfs", "ext4", "ext3", "ext2", "f2fs", "reiserfs", "jfs", "btrfs",
		"fuseblk", "zfs", "simfs", "ntfs", "fat32", "exfat", "xfs", "fuse.rclone",
	}
	excludeNetInterfaces = []string{
		"lo", "tun", "docker", "veth", "br-", "vmbr", "vnet", "kube",
	}
)

var (
	netInSpeed, netOutSpeed, netInTransfer, netOutTransfer, lastUpdateNetStats uint64
	cachedBootTime                                                             time.Time
)

// GetHost 获取主机硬件信息
func GetHost(agentConfig *model.AgentConfig) *model.Host {
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
	diskTotal, _ := getDiskTotalAndUsed(agentConfig)

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

func GetState(agentConfig *model.AgentConfig, skipConnectionCount bool, skipProcsCount bool) *model.HostState {
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

	_, diskUsed := getDiskTotalAndUsed(agentConfig)
	loadStat, _ := load.Avg()

	var tcpConnCount, udpConnCount uint64
	if !skipConnectionCount {
		ss_err := true
		if runtime.GOOS == "linux" {
			tcpStat, err_tcp := goss.ConnectionsWithProtocol(syscall.IPPROTO_TCP)
			udpStat, err_udp := goss.ConnectionsWithProtocol(syscall.IPPROTO_UDP)
			if err_tcp == nil && err_udp == nil {
				ss_err = false
				tcpConnCount = uint64(len(tcpStat))
				udpConnCount = uint64(len(udpStat))
			}
		}
		if ss_err {
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

// TrackNetworkSpeed NIC监控，统计流量与速度
func TrackNetworkSpeed(agentConfig *model.AgentConfig) {
	var innerNetInTransfer, innerNetOutTransfer uint64
	nc, err := net.IOCounters(true)
	if err == nil {
		for _, v := range nc {
			if len(agentConfig.NICAllowlist) > 0 {
				if !agentConfig.NICAllowlist[v.Name] {
					continue
				}
			} else {
				if isListContainsStr(excludeNetInterfaces, v.Name) {
					continue
				}
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

func getDiskTotalAndUsed(agentConfig *model.AgentConfig) (total uint64, used uint64) {
	devices := make(map[string]string)

	if len(agentConfig.HardDrivePartitionAllowlist) > 0 {
		// 如果配置了白名单，使用白名单的列表
		for i, v := range agentConfig.HardDrivePartitionAllowlist {
			devices[strconv.Itoa(i)] = v
		}
	} else {
		// 否则使用默认过滤规则
		diskList, _ := disk.Partitions(false)
		for _, d := range diskList {
			fsType := strings.ToLower(d.Fstype)
			// 不统计 K8s 的虚拟挂载点：https://github.com/shirou/gopsutil/issues/1007
			if devices[d.Device] == "" && isListContainsStr(expectDiskFsTypes, fsType) && !strings.Contains(d.Mountpoint, "/var/lib/kubelet") {
				devices[d.Device] = d.Mountpoint
			}
		}
	}

	for _, mountPath := range devices {
		diskUsageOf, err := disk.Usage(mountPath)
		if err == nil {
			total += diskUsageOf.Total
			used += diskUsageOf.Used
		}
	}

	// Fallback 到这个方法,仅统计根路径,适用于OpenVZ之类的.
	if runtime.GOOS == "linux" && total == 0 && used == 0 {
		cmd := exec.Command("df")
		out, err := cmd.CombinedOutput()
		if err == nil {
			s := strings.Split(string(out), "\n")
			for _, c := range s {
				info := strings.Fields(c)
				if len(info) == 6 {
					if info[5] == "/" {
						total, _ = strconv.ParseUint(info[1], 0, 64)
						used, _ = strconv.ParseUint(info[2], 0, 64)
						// 默认获取的是1K块为单位的.
						total = total * 1024
						used = used * 1024
					}
				}
			}
		}
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
