package monitor

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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

type ipDotSbGeoIP struct {
	CountryCode string `json:"country_code,omitempty"`
	IP          string `json:"ip,omitempty"`
}

var netInSpeed, netOutSpeed, netInTransfer, netOutTransfer, lastUpdate uint64

func GetHost() *model.Host {
	hi, _ := host.Info()
	var cpus []string
	ci, _ := cpu.Info()
	for i := 0; i < len(ci); i++ {
		cpus = append(cpus, fmt.Sprintf("%v-%vC%vT", ci[i].ModelName, ci[i].Cores, ci[i].Stepping))
	}
	mv, _ := mem.VirtualMemory()
	ms, _ := mem.SwapMemory()
	u, _ := disk.Usage("/")
	var ip ipDotSbGeoIP
	resp, err := http.Get("https://api-ipv4.ip.sb/geoip")
	if err == nil {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		json.Unmarshal(body, &ip)
	}
	resp, err = http.Get("https://api-ipv6.ip.sb/ip")
	if err == nil {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		ip.IP = fmt.Sprintf("ip(v4: %s, v6: %s)", ip.IP, body)
	}
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
		IP:              ip.IP,
		CountryCode:     strings.ToLower(ip.CountryCode),
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
