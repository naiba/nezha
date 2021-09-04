package model

import (
	"encoding/json"
	"fmt"
	"html/template"
	"time"

	pb "github.com/naiba/nezha/proto"
)

type Server struct {
	Common
	Name         string
	Tag          string // 分组名
	Secret       string `gorm:"uniqueIndex" json:"-"`
	Note         string `json:"-"` // 管理员可见备注
	DisplayIndex int    // 展示排序，越大越靠前

	Host       *Host      `gorm:"-"`
	State      *HostState `gorm:"-"`
	LastActive time.Time  `gorm:"-"`

	TaskClose  chan error                        `gorm:"-" json:"-"`
	TaskStream pb.NezhaService_RequestTaskServer `gorm:"-" json:"-"`

	PrevHourlyTransferIn  int64 `gorm:"-" json:"-"` // 上次数据点时的入站使用量
	PrevHourlyTransferOut int64 `gorm:"-" json:"-"` // 上次数据点时的出站使用量
}

func (s *Server) CopyFromRunningServer(old *Server) {
	s.Host = old.Host
	s.State = old.State
	s.LastActive = old.LastActive
	s.TaskClose = old.TaskClose
	s.TaskStream = old.TaskStream
	s.PrevHourlyTransferIn = old.PrevHourlyTransferIn
	s.PrevHourlyTransferOut = old.PrevHourlyTransferOut
}

func (s Server) Marshal() template.JS {
	name, _ := json.Marshal(s.Name)
	tag, _ := json.Marshal(s.Tag)
	note, _ := json.Marshal(s.Note)
	secret, _ := json.Marshal(s.Secret)
	return template.JS(fmt.Sprintf(`{"ID":%d,"Name":%s,"Secret":%s,"DisplayIndex":%d,"Tag":%s,"Note":%s}`, s.ID, name, secret, s.DisplayIndex, tag, note)) // #nosec
}
