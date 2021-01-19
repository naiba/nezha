package model

import (
	pb "github.com/naiba/nezha/proto"
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
	Name   string
	Type   uint8
	Target string
}

func (m *Monitor) PB() *pb.Task {
	return &pb.Task{
		Id:   m.ID,
		Type: uint64(m.Type),
		Data: m.Target,
	}
}
