package model

import (
	"encoding/json"

	pb "github.com/naiba/nezha/proto"
	"gorm.io/gorm"
)

const (
	_ = iota
	TaskTypeHTTPGET
	TaskTypeICMPPing
	TaskTypeTCPPing
	TaskTypeCommand
	TaskTypeTerminal
)

type TerminalTask struct {
	// websocket 主机名
	Host string `json:"host,omitempty"`
	// 是否启用 SSL
	UseSSL bool `json:"use_ssl,omitempty"`
	// 会话标识
	Session string `json:"session,omitempty"`
}

const (
	MonitorCoverAll = iota
	MonitorCoverIgnoreAll
)

type Monitor struct {
	Common
	Name           string
	Type           uint8
	Target         string
	SkipServersRaw string
	Notify         bool
	Cover          uint8
	SkipServers    map[uint64]bool `gorm:"-" json:"-"`
}

func (m *Monitor) PB() *pb.Task {
	return &pb.Task{
		Id:   m.ID,
		Type: uint64(m.Type),
		Data: m.Target,
	}
}

func (m *Monitor) AfterFind(tx *gorm.DB) error {
	var skipServers []uint64
	if err := json.Unmarshal([]byte(m.SkipServersRaw), &skipServers); err != nil {
		return err
	}
	m.SkipServers = make(map[uint64]bool)
	for i := 0; i < len(skipServers); i++ {
		m.SkipServers[skipServers[i]] = true
	}
	return nil
}
