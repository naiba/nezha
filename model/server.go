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
	Tag          string // 分组名
	Secret       string `json:"-"`
	Note         string `json:"-"` // 管理员可见备注
	DisplayIndex int    // 展示权重，越大越靠前

	Host       *Host      `gorm:"-"`
	State      *HostState `gorm:"-"`
	LastActive time.Time  `gorm:"-"`

	TaskClose  chan error                        `gorm:"-" json:"-"`
	TaskStream pb.NezhaService_RequestTaskServer `gorm:"-" json:"-"`
}

func (s Server) Marshal() template.JS {
	return template.JS(fmt.Sprintf(`{"ID":%d,"Name":"%s","Secret":"%s","DisplayIndex":%d,"Tag":"%s","Note":"%s"}`, s.ID, s.Name, s.Secret, s.DisplayIndex, s.Tag, s.Note))
}
