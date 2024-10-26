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
	Name                string    `json:"name,omitempty"`
	TaskType            uint8     `gorm:"default:0" json:"task_type,omitempty"` // 0:计划任务 1:触发任务
	Scheduler           string    `json:"scheduler,omitempty"`                  // 分钟 小时 天 月 星期
	Command             string    `json:"command,omitempty"`
	Servers             []uint64  `gorm:"-" json:"servers,omitempty"`
	PushSuccessful      bool      `json:"push_successful,omitempty"`       // 推送成功的通知
	NotificationGroupID uint64    `json:"notification_group_id,omitempty"` // 指定通知方式的分组
	LastExecutedAt      time.Time `json:"last_executed_at,omitempty"`      // 最后一次执行时间
	LastResult          bool      `json:"last_result,omitempty"`           // 最后一次执行结果
	Cover               uint8     `json:"cover,omitempty"`                 // 计划任务覆盖范围 (0:仅覆盖特定服务器 1:仅忽略特定服务器 2:由触发该计划任务的服务器执行)

	CronJobID  cron.EntryID `gorm:"-" json:"cron_job_id,omitempty"`
	ServersRaw string       `json:"-"`
}

func (c *Cron) AfterFind(tx *gorm.DB) error {
	return utils.Json.Unmarshal([]byte(c.ServersRaw), &c.Servers)
}
