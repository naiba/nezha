package model

import (
	"time"

	"github.com/naiba/nezha/pkg/utils"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

const (
	CronCoverIgnoreAll = iota
	CronCoverAll
	CronCoverAlertTrigger
	CronTypeCronTask    = 0
	CronTypeTriggerTask = 1
)

type Cron struct {
	Common
	Name            string
	TaskType        uint8  `gorm:"default:0"` // 0:计划任务 1:触发任务
	Scheduler       string //分钟 小时 天 月 星期
	Command         string
	Servers         []uint64  `gorm:"-"`
	PushSuccessful  bool      // 推送成功的通知
	NotificationTag string    // 指定通知方式的分组
	LastExecutedAt  time.Time // 最后一次执行时间
	LastResult      bool      // 最后一次执行结果
	Cover           uint8     // 计划任务覆盖范围 (0:仅覆盖特定服务器 1:仅忽略特定服务器 2:由触发该计划任务的服务器执行)

	CronJobID  cron.EntryID `gorm:"-"`
	ServersRaw string
}

func (c *Cron) AfterFind(tx *gorm.DB) error {
	return utils.Json.Unmarshal([]byte(c.ServersRaw), &c.Servers)
}
