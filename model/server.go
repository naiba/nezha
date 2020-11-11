package model

import (
	"time"

	pb "github.com/naiba/nezha/proto"
)

// Server ..
type Server struct {
	Common
	Name   string
	Secret string

	Host       *Host
	State      *State
	LastActive time.Time

	Stream      pb.NezhaService_HeartbeatServer `gorm:"-" json:"-"`
	StreamClose chan<- error                    `gorm:"-" json:"-"`
}
