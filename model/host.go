package model

import (
	"fmt"

	pb "github.com/naiba/nezha/proto"
)

const (
	_ = iota

	MTReportHostState
)

type HostState struct {
	CPU            float64
	MemUsed        uint64
	SwapUsed       uint64
	DiskUsed       uint64
	NetInTransfer  uint64
	NetOutTransfer uint64
	NetInSpeed     uint64
	NetOutSpeed    uint64
	Uptime         uint64
}

func (s *HostState) PB() *pb.State {
	return &pb.State{
		Cpu:            s.CPU,
		MemUsed:        s.MemUsed,
		SwapUsed:       s.SwapUsed,
		DiskUsed:       s.DiskUsed,
		NetInTransfer:  s.NetInTransfer,
		NetOutTransfer: s.NetOutTransfer,
		NetInSpeed:     s.NetInSpeed,
		NetOutSpeed:    s.NetOutSpeed,
		Uptime:         s.Uptime,
	}
}

func PB2State(s *pb.State) HostState {
	return HostState{
		CPU:            s.GetCpu(),
		MemUsed:        s.GetMemUsed(),
		SwapUsed:       s.GetSwapUsed(),
		DiskUsed:       s.GetDiskUsed(),
		NetInTransfer:  s.GetNetInTransfer(),
		NetOutTransfer: s.GetNetOutTransfer(),
		NetInSpeed:     s.GetNetInSpeed(),
		NetOutSpeed:    s.GetNetOutSpeed(),
		Uptime:         s.GetUptime(),
	}
}

type Host struct {
	Platform        string
	PlatformVersion string
	CPU             []string
	MemTotal        uint64
	DiskTotal       uint64
	SwapTotal       uint64
	Arch            string
	Virtualization  string
	BootTime        uint64
	IP              string `json:"-"`
	CountryCode     string
	Version         string
}

func (h *Host) PB() *pb.Host {
	return &pb.Host{
		Platform:        h.Platform,
		PlatformVersion: h.PlatformVersion,
		Cpu:             h.CPU,
		MemTotal:        h.MemTotal,
		DiskTotal:       h.DiskTotal,
		SwapTotal:       h.SwapTotal,
		Arch:            h.Arch,
		Virtualization:  h.Virtualization,
		BootTime:        h.BootTime,
		Ip:              h.IP,
		CountryCode:     h.CountryCode,
		Version:         h.Version,
	}
}

func PB2Host(h *pb.Host) Host {

	cpuCount := make(map[string]int, 0)
	cpus := h.GetCpu()
	for _, u := range cpus {
		cpuCount[u]++
	}

	var distCpu []string
	for u, num := range cpuCount {
		distCpu = append(distCpu, fmt.Sprintf("%sx%d", u, num))
	}

	return Host{
		Platform:        h.GetPlatform(),
		PlatformVersion: h.GetPlatformVersion(),
		CPU:             distCpu,
		MemTotal:        h.GetMemTotal(),
		DiskTotal:       h.GetDiskTotal(),
		SwapTotal:       h.GetSwapTotal(),
		Arch:            h.GetArch(),
		Virtualization:  h.GetVirtualization(),
		BootTime:        h.GetBootTime(),
		IP:              h.GetIp(),
		CountryCode:     h.GetCountryCode(),
		Version:         h.GetVersion(),
	}
}
