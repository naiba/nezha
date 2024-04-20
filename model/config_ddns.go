package model

import (
	"os"

	"github.com/spf13/viper"
	"sigs.k8s.io/yaml"
)

type ConfigDDNS struct {
	v                  *viper.Viper

	IgnoredIPNotification      string // 特定服务器IP（多个服务器用逗号分隔）
	IgnoredIPNotificationServerIDs map[uint64]bool // [ServerID] -> bool(值为true代表当前ServerID在特定服务器列表内）
	Profiles map[string]ProfileConfig
}


type ProfileConfig struct {
    Provider           string
    AccessID           string
    AccessSecret       string
    WebhookURL         string
    WebhookMethod      string
    WebhookRequestBody string
    WebhookHeaders     string
}


// Read 读取配置文件并应用
func (c *ConfigDDNS) Read(path string) error {
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

	return nil
}

// Save 保存配置文件
func (c *ConfigDDNS) Save() error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(c.v.ConfigFileUsed(), data, os.ModePerm)
}