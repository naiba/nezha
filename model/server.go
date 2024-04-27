package model

import (
	"fmt"
	"html/template"
	"time"

	"github.com/naiba/nezha/pkg/utils"
	pb "github.com/naiba/nezha/proto"
)

type Server struct {
	Common
	Name         string
	Tag          string // 分组名
	Secret       string `gorm:"uniqueIndex" json:"-"`
	Note         string `json:"-"` // 管理员可见备注
	DisplayIndex int    // 展示排序，越大越靠前
	HideForGuest bool   // 对游客隐藏
	EnableDDNS   bool   // 是否启用DDNS 未在配置文件中启用DDNS 或 DDNS检查时间为0时此项无效
	EnableIPv4   bool   // 是否启用DDNS IPv4
	EnableIpv6   bool   // 是否启用DDNS IPv6
	DDNSDomain   string // DDNS中的前缀 如基础域名为abc.oracle DDNSName为mjj 就会把mjj.abc.oracle解析服务器IP 为空则停用
	DDNSProfile  string // DDNS配置

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

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func (s Server) Marshal() template.JS {
	name, _ := utils.Json.Marshal(s.Name)
	tag, _ := utils.Json.Marshal(s.Tag)
	note, _ := utils.Json.Marshal(s.Note)
	secret, _ := utils.Json.Marshal(s.Secret)
	ddnsDomain, _ := utils.Json.Marshal(s.DDNSDomain)
	ddnsProfile, _ := utils.Json.Marshal(s.DDNSProfile)
	return template.JS(fmt.Sprintf(`{"ID":%d,"Name":%s,"Secret":%s,"DisplayIndex":%d,"Tag":%s,"Note":%s,"HideForGuest": %s,"EnableDDNS": %s,"EnableIPv4": %s,"EnableIpv6": %s,"DDNSDomain": %s,"DDNSProfile": %s}`, s.ID, name, secret, s.DisplayIndex, tag, note, boolToString(s.HideForGuest), boolToString(s.EnableDDNS), boolToString(s.EnableIPv4), boolToString(s.EnableIpv6), ddnsDomain, ddnsProfile)) // #nosec
}
