package model

import "time"

type StreamServer struct {
	ID           uint64 `json:"id,omitempty"`
	Name         string `json:"name,omitempty"`
	PublicNote   string `json:"public_note,omitempty"`   // 公开备注，只第一个数据包有值
	DisplayIndex int    `json:"display_index,omitempty"` // 展示排序，越大越靠前

	Host       *Host      `json:"host,omitempty"`
	State      *HostState `json:"state,omitempty"`
	LastActive time.Time  `json:"last_active,omitempty"`
}

type StreamServerData struct {
	Now     int64          `json:"now,omitempty"`
	Servers []StreamServer `json:"servers,omitempty"`
}

type ServerForm struct {
	Name         string   `json:"name,omitempty"`
	Note         string   `json:"note,omitempty"`                   // 管理员可见备注
	PublicNote   string   `json:"public_note,omitempty"`            // 公开备注
	DisplayIndex int      `json:"display_index,omitempty"`          // 展示排序，越大越靠前
	HideForGuest bool     `json:"hide_for_guest,omitempty"`         // 对游客隐藏
	EnableDDNS   bool     `json:"enable_ddns,omitempty"`            // 启用DDNS
	DDNSProfiles []uint64 `gorm:"-" json:"ddns_profiles,omitempty"` // DDNS配置
}
