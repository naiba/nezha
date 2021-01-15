package model

import (
	"fmt"
	"html/template"
	"time"

	pb "github.com/naiba/nezha/proto"
)

type Server struct {
	Common
	Name         string
	DisplayIndex int    // 展示权重，越大越靠前
	Secret       string `json:"-"`
	Tag          string
	Host         *Host      `gorm:"-"`
	State        *HostState `gorm:"-"`
	LastActive   time.Time

	TaskClose  chan error                        `gorm:"-" json:"-"`
	TaskStream pb.NezhaService_RequestTaskServer `gorm:"-" json:"-"`
}

func (s Server) Marshal() template.JS {
	return template.JS(fmt.Sprintf(`{"ID":%d,"Name":"%s","Secret":"%s"}`, s.ID, s.Name, s.Secret))
}
