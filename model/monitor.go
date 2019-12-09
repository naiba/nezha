package model

import pb "github.com/p14yground/nezha/proto"

const (
	_ = iota
	// MTReportState ..
	MTReportState
)

// State ..
type State struct {
	CPU       float64
	MEMTotal  uint64
	MEMUsed   uint64
	SwapTotal uint64
	SwapUsed  uint64
	DiskTotal uint64
	DiskUsed  uint64
	NetIn     uint64
	NetOut    uint64
}

// PB ..
func (s *State) PB() *pb.State {
	return &pb.State{
		Cpu:       s.CPU,
		MemTotal:  s.MEMTotal,
		MemUsed:   s.MEMUsed,
		SwapTotal: s.SwapTotal,
		SwapUsed:  s.SwapUsed,
		DiskTotal: s.DiskTotal,
		DiskUsed:  s.DiskUsed,
		NetIn:     s.NetIn,
		NetOut:    s.NetOut,
	}
}

// Host ..
type Host struct {
	Platform        string
	PlatformVersion string
	CPU             []string
	Arch            string
	Virtualization  string
	Uptime          string
	BootTime        string
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
		Uptime:          h.Uptime,
		BootTime:        h.BootTime,
		Ip:              h.IP,
		CountryCode:     h.CountryCode,
		Version:         h.Version,
	}
}
