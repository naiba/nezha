package model

import (
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

const (
	ConfigTypeGitHub = "github"
	ConfigTypeGitee  = "gitee"
)

const (
	ConfigCoverAll = iota
	ConfigCoverIgnoreAll
)

type Config struct {
	Debug bool
	Site  struct {
		Brand        string // 站点名称
		CookieName   string // 浏览器 Cookie 名称
		Theme        string
		CustomCode   string
		ViewPassword string // 前台查看密码
	}
	Oauth2 struct {
		Type         string
		Admin        string // 管理员用户名列表
		ClientID     string
		ClientSecret string
	}
	HTTPPort                   uint
	GRPCPort                   uint
	GRPCHost                   string
	EnableIPChangeNotification bool

	// IP变更提醒
	Cover                 uint8  // 覆盖范围
	IgnoredIPNotification string // 特定服务器

	v                              *viper.Viper
	IgnoredIPNotificationServerIDs map[uint64]bool
}

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

	if c.Site.Theme == "" {
		c.Site.Theme = "default"
	}

	c.updateIgnoredIPNotificationID()
	return nil
}

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

func (c *Config) Save() error {
	c.updateIgnoredIPNotificationID()
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(c.v.ConfigFileUsed(), data, os.ModePerm)
}
