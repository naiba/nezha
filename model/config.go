package model

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Config ..
type Config struct {
	Debug bool
	Site  struct {
		Brand      string // 站点名称
		CookieName string // 浏览器 Cookie 名称
	}
	GitHub struct {
		Admin        []int64 // 管理员ID列表
		ClientID     string
		ClientSecret string
	}

	v *viper.Viper
}

// ReadInConfig ..
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

	c.v.OnConfigChange(func(in fsnotify.Event) {
		fmt.Println("配置文件更新，重载配置")
		c.v.Unmarshal(c)
	})

	go c.v.WatchConfig()
	return nil
}
