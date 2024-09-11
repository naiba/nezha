package model

type ApiToken struct {
	Common
	UserID uint64 `json:"user_id"`
	Token  string `json:"token"`
	Note   string `json:"note"`
	Scope  Scope  `json:"scope" gorm:"serializer:json"`
}

type Permission struct {
	ReadAll  bool     `json:"read_all"`
	WriteAll bool     `json:"write_all"`
	Read     []uint64 `json:"read"`
	Write    []uint64 `json:"write"`
}

type Scope struct {
	// 状态信息
	ServerStatus  Permission `json:"server_status"`
	ServiceStatus Permission `json:"service_status"`

	// 配置信息
	Server  Permission `json:"server"`
	Service Permission `json:"service"`
	Cron    Permission `json:"cron"`
}
