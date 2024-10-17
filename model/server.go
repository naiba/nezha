package model

import (
	"fmt"
	"html/template"
	"log"
	"sync"
	"time"

	"github.com/naiba/nezha/pkg/utils"
	pb "github.com/naiba/nezha/proto"
	"gorm.io/gorm"
)

type Server struct {
	Common
	Name         string
	Tag          string   // 分组名
	Secret       string   `gorm:"uniqueIndex" json:"-"`
	Note         string   `json:"-"`                    // 管理员可见备注
	PublicNote   string   `json:"PublicNote,omitempty"` // 公开备注
	DisplayIndex int      // 展示排序，越大越靠前
	HideForGuest bool     // 对游客隐藏
	EnableDDNS   bool     // 启用DDNS
	DDNSProfiles []uint64 `gorm:"-" json:"-"` // DDNS配置

	DDNSProfilesRaw string `gorm:"default:'[]';column:ddns_profiles_raw" json:"-"`

	Host       *Host      `gorm:"-"`
	State      *HostState `gorm:"-"`
	LastActive time.Time  `gorm:"-"`

	TaskClose     chan error                        `gorm:"-" json:"-"`
	TaskCloseLock *sync.Mutex                       `gorm:"-" json:"-"`
	TaskStream    pb.NezhaService_RequestTaskServer `gorm:"-" json:"-"`

	PrevTransferInSnapshot  int64 `gorm:"-" json:"-"` // 上次数据点时的入站使用量
	PrevTransferOutSnapshot int64 `gorm:"-" json:"-"` // 上次数据点时的出站使用量
}

func (s *Server) CopyFromRunningServer(old *Server) {
	s.Host = old.Host
	s.State = old.State
	s.LastActive = old.LastActive
	s.TaskClose = old.TaskClose
	s.TaskCloseLock = old.TaskCloseLock
	s.TaskStream = old.TaskStream
	s.PrevTransferInSnapshot = old.PrevTransferInSnapshot
	s.PrevTransferOutSnapshot = old.PrevTransferOutSnapshot
}

func (s *Server) AfterFind(tx *gorm.DB) error {
	if s.DDNSProfilesRaw != "" {
		if err := utils.Json.Unmarshal([]byte(s.DDNSProfilesRaw), &s.DDNSProfiles); err != nil {
			log.Println("NEZHA>> Server.AfterFind:", err)
			return nil
		}
	}
	return nil
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func (s Server) MarshalForDashboard() template.JS {
	name, _ := utils.Json.Marshal(s.Name)
	tag, _ := utils.Json.Marshal(s.Tag)
	note, _ := utils.Json.Marshal(s.Note)
	secret, _ := utils.Json.Marshal(s.Secret)
	ddnsProfilesRaw, _ := utils.Json.Marshal(s.DDNSProfilesRaw)
	publicNote, _ := utils.Json.Marshal(s.PublicNote)
	return template.JS(fmt.Sprintf(`{"ID":%d,"Name":%s,"Secret":%s,"DisplayIndex":%d,"Tag":%s,"Note":%s,"HideForGuest": %s,"EnableDDNS": %s,"DDNSProfilesRaw": %s,"PublicNote": %s}`, s.ID, name, secret, s.DisplayIndex, tag, note, boolToString(s.HideForGuest), boolToString(s.EnableDDNS), ddnsProfilesRaw, publicNote))
}
