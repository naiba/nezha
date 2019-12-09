package model

import pb "github.com/p14yground/nezha/proto"

const (
	_ = iota
	// MTReportState ..
	MTReportState
)

// State ..
type State struct {
	CPU            float64
	MemTotal       uint64
	MemUsed        uint64
	SwapTotal      uint64
	SwapUsed       uint64
	DiskTotal      uint64
	DiskUsed       uint64
	NetInTransfer  uint64
	NetOutTransfer uint64
	NetInSpeed     uint64
	NetOutSpeed    uint64
	Uptime         uint64
}

// PB ..
func (s *State) PB() *pb.State {
	return &pb.State{
		Cpu:            s.CPU,
		MemTotal:       s.MemTotal,
		MemUsed:        s.MemUsed,
		SwapTotal:      s.SwapTotal,
		SwapUsed:       s.SwapUsed,
		DiskTotal:      s.DiskTotal,
		DiskUsed:       s.DiskUsed,
		NetInTransfer:  s.NetInTransfer,
		NetOutTransfer: s.NetOutTransfer,
		NetInSpeed:     s.NetInSpeed,
		NetOutSpeed:    s.NetOutSpeed,
		Uptime:         s.Uptime,
	}
}

// PB2State ..
func PB2State(s *pb.State) State {
	return State{
		CPU:            s.GetCpu(),
		MemTotal:       s.GetMemTotal(),
		MemUsed:        s.GetMemUsed(),
		SwapTotal:      s.GetSwapTotal(),
		SwapUsed:       s.GetSwapUsed(),
		DiskTotal:      s.GetDiskTotal(),
		DiskUsed:       s.GetDiskUsed(),
		NetInTransfer:  s.GetNetInTransfer(),
		NetOutTransfer: s.GetNetOutTransfer(),
		NetInSpeed:     s.GetNetInSpeed(),
		NetOutSpeed:    s.GetNetOutSpeed(),
		Uptime:         s.GetUptime(),
	}
}

// Host ..
type Host struct {
	Platform        string
	PlatformVersion string
	CPU             []string
	Arch            string
	Virtualization  string
	BootTime        uint64
	IP              string
	CountryCode     string
	Version         string
}

// PB ..
func (h *Host) PB() *pb.Host {
	return &pb.Host{
		Platform:        h.Platform,
		PlatformVersion: h.PlatformVersion,
		Cpu:             h.CPU,
		Arch:            h.Arch,
		Virtualization:  h.Virtualization,
		BootTime:        h.BootTime,
		Ip:              h.IP,
		CountryCode:     h.CountryCode,
		Version:         h.Version,
	}
}

// PB2Host ...
func PB2Host(h *pb.Host) Host {
	return Host{
		Platform:        h.GetPlatform(),
		PlatformVersion: h.GetPlatformVersion(),
		CPU:             h.GetCpu(),
		Arch:            h.GetArch(),
		Virtualization:  h.GetVirtualization(),
		BootTime:        h.GetBootTime(),
		IP:              h.GetIp(),
		CountryCode:     h.GetCountryCode(),
		Version:         h.GetVersion(),
	}
}
