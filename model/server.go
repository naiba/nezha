package model

import (
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/nezhahq/nezha/pkg/utils"
	pb "github.com/nezhahq/nezha/proto"
)

type Server struct {
	Common

	Name            string `json:"name"`
	UUID            string `json:"uuid,omitempty" gorm:"unique"`
	Note            string `json:"note,omitempty"`           // 管理员可见备注
	PublicNote      string `json:"public_note,omitempty"`    // 公开备注
	DisplayIndex    int    `json:"display_index"`            // 展示排序，越大越靠前
	HideForGuest    bool   `json:"hide_for_guest,omitempty"` // 对游客隐藏
	EnableDDNS      bool   `json:"enable_ddns,omitempty"`    // 启用DDNS
	DDNSProfilesRaw string `gorm:"default:'[]';column:ddns_profiles_raw" json:"-"`

	DDNSProfiles []uint64 `gorm:"-" json:"ddns_profiles,omitempty" validate:"optional"` // DDNS配置

	Host       *Host      `gorm:"-" json:"host,omitempty"`
	State      *HostState `gorm:"-" json:"state,omitempty"`
	GeoIP      *GeoIP     `gorm:"-" json:"geoip,omitempty"`
	LastActive time.Time  `gorm:"-" json:"last_active,omitempty"`

	TaskStream pb.NezhaService_RequestTaskServer `gorm:"-" json:"-"`

	PrevTransferInSnapshot  int64 `gorm:"-" json:"-"` // 上次数据点时的入站使用量
	PrevTransferOutSnapshot int64 `gorm:"-" json:"-"` // 上次数据点时的出站使用量
}

func (s *Server) CopyFromRunningServer(old *Server) {
	s.Host = old.Host
	s.State = old.State
	s.GeoIP = old.GeoIP
	s.LastActive = old.LastActive
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
