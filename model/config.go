package model

import (
	"os"
	"strconv"
	"strings"

	"github.com/naiba/nezha/pkg/utils"
	"github.com/spf13/viper"
	"sigs.k8s.io/yaml"
)

const (
	ConfigCoverAll = iota
	ConfigCoverIgnoreAll
)

type Config struct {
	Debug bool `mapstructure:"debug" json:"debug,omitempty"` // debug模式开关

	Language       string `mapstructure:"language" json:"language,omitempty"` // 系统语言，默认 zh-CN
	SiteName       string `mapstructure:"site_name" json:"site_name,omitempty"`
	JWTSecretKey   string `mapstructure:"jwt_secret_key" json:"jwt_secret_key,omitempty"`
	AgentSecretKey string `mapstructure:"agent_secret_key" json:"agent_secret_key,omitempty"`
	ListenPort     uint   `mapstructure:"listen_port" json:"listen_port,omitempty"`
	InstallHost    string `mapstructure:"install_host" json:"install_host,omitempty"`
	TLS            bool   `mapstructure:"tls" json:"tls,omitempty"`
	Location       string `mapstructure:"location" json:"location,omitempty"` // 时区，默认为 Asia/Shanghai

	EnablePlainIPInNotification bool `mapstructure:"enable_plain_ip_in_notification" json:"enable_plain_ip_in_notification,omitempty"` // 通知信息IP不打码

	// IP变更提醒
	EnableIPChangeNotification  bool   `mapstructure:"enable_ip_change_notification" json:"enable_ip_change_notification,omitempty"`
	IPChangeNotificationGroupID uint64 `mapstructure:"ip_change_notification_group_id" json:"ip_change_notification_group_id,omitempty"`
	Cover                       uint8  `mapstructure:"cover" json:"cover,omitempty"`                                     // 覆盖范围（0:提醒未被 IgnoredIPNotification 包含的所有服务器; 1:仅提醒被 IgnoredIPNotification 包含的服务器;）
	IgnoredIPNotification       string `mapstructure:"ignored_ip_notification" json:"ignored_ip_notification,omitempty"` // 特定服务器IP（多个服务器用逗号分隔）

	IgnoredIPNotificationServerIDs map[uint64]bool `mapstructure:"ignored_ip_notification_server_ids" json:"ignored_ip_notification_server_ids,omitempty"` // [ServerID] -> bool(值为true代表当前ServerID在特定服务器列表内）
	AvgPingCount                   int             `mapstructure:"avg_ping_count" json:"avg_ping_count,omitempty"`
	DNSServers                     string          `mapstructure:"dns_servers" json:"dns_servers,omitempty"`

	CustomCode          string `mapstructure:"custom_code" json:"custom_code,omitempty"`
	CustomCodeDashboard string `mapstructure:"custom_code_dashboard" json:"custom_code_dashboard,omitempty"`

	v *viper.Viper `json:"-"`
}

// Read 读取配置文件并应用
func (c *Config) Read(path string) error {
	c.v = viper.New()
	c.v.SetConfigFile(path)
	err := c.v.ReadInConfig()
	if err != nil {
		return err
	}

	err = c.v.Unmarshal(c)
	if err != nil {
		return err
	}

	if c.ListenPort == 0 {
		c.ListenPort = 8008
	}
	if c.Location == "" {
		c.Location = "Asia/Shanghai"
	}
	if c.AvgPingCount == 0 {
		c.AvgPingCount = 2
	}
	if c.JWTSecretKey == "" {
		c.JWTSecretKey, err = utils.GenerateRandomString(1024)
		if err != nil {
			return err
		}
		if err = c.Save(); err != nil {
			return err
		}
	}

	if c.AgentSecretKey == "" {
		c.AgentSecretKey, err = utils.GenerateRandomString(32)
		if err != nil {
			return err
		}
		if err = c.Save(); err != nil {
			return err
		}
	}

	c.updateIgnoredIPNotificationID()
	return nil
}

// updateIgnoredIPNotificationID 更新用于判断服务器ID是否属于特定服务器的map
func (c *Config) updateIgnoredIPNotificationID() {
	c.IgnoredIPNotificationServerIDs = make(map[uint64]bool)
	splitedIDs := strings.Split(c.IgnoredIPNotification, ",")
	for i := 0; i < len(splitedIDs); i++ {
		id, _ := strconv.ParseUint(splitedIDs[i], 10, 64)
		if id > 0 {
			c.IgnoredIPNotificationServerIDs[id] = true
		}
	}
}

// Save 保存配置文件
func (c *Config) Save() error {
	c.updateIgnoredIPNotificationID()
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(c.v.ConfigFileUsed(), data, 0600)
}
