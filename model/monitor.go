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
)

type Monitor struct {
	Common
	Name           string
	Type           uint8
	Target         string
	SkipServersRaw string
	Notify         bool

	SkipServers map[uint64]bool `gorm:"-" json:"-"`
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
