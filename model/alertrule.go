package model

import (
	"time"

	"github.com/naiba/nezha/pkg/utils"
	"gorm.io/gorm"
)

const (
	ModeAlwaysTrigger  = 0
	ModeOnetimeTrigger = 1
)

type CycleTransferStats struct {
	Name       string
	From       time.Time
	To         time.Time
	Max        uint64
	Min        uint64
	ServerName map[uint64]string
	Transfer   map[uint64]uint64
	NextUpdate map[uint64]time.Time
}

type AlertRule struct {
	Common
	Name                   string
	RulesRaw               string
	Enable                 *bool
	TriggerMode            int      `gorm:"default:0"` // 触发模式: 0-始终触发(默认) 1-单次触发
	NotificationTag        string   // 该报警规则所在的通知组
	FailTriggerTasksRaw    string   `gorm:"default:'[]'"`
	RecoverTriggerTasksRaw string   `gorm:"default:'[]'"`
	Rules                  []Rule   `gorm:"-" json:"-"`
	FailTriggerTasks       []uint64 `gorm:"-" json:"-"` // 失败时执行的触发任务id
	RecoverTriggerTasks    []uint64 `gorm:"-" json:"-"` // 恢复时执行的触发任务id
}

func (r *AlertRule) BeforeSave(tx *gorm.DB) error {
	if data, err := utils.Json.Marshal(r.Rules); err != nil {
		return err
	} else {
		r.RulesRaw = string(data)
	}
	if data, err := utils.Json.Marshal(r.FailTriggerTasks); err != nil {
		return err
	} else {
		r.FailTriggerTasksRaw = string(data)
	}
	if data, err := utils.Json.Marshal(r.RecoverTriggerTasks); err != nil {
		return err
	} else {
		r.RecoverTriggerTasksRaw = string(data)
	}
	return nil
}

func (r *AlertRule) AfterFind(tx *gorm.DB) error {
	var err error
	if err = utils.Json.Unmarshal([]byte(r.RulesRaw), &r.Rules); err != nil {
		return err
	}
	if err = utils.Json.Unmarshal([]byte(r.FailTriggerTasksRaw), &r.FailTriggerTasks); err != nil {
		return err
	}
	if err = utils.Json.Unmarshal([]byte(r.RecoverTriggerTasksRaw), &r.RecoverTriggerTasks); err != nil {
		return err
	}
	return nil
}

func (r *AlertRule) Enabled() bool {
	return r.Enable != nil && *r.Enable
}

// Snapshot 对传入的Server进行该报警规则下所有type的检查 返回包含每项检查结果的空接口
func (r *AlertRule) Snapshot(cycleTransferStats *CycleTransferStats, server *Server, db *gorm.DB) []interface{} {
	var point []interface{}
	for i := 0; i < len(r.Rules); i++ {
		point = append(point, r.Rules[i].Snapshot(cycleTransferStats, server, db))
	}
	return point
}

// Check 传入包含当前报警规则下所有type检查结果的空接口 返回报警持续时间与是否通过报警检查(通过则返回true)
func (r *AlertRule) Check(points [][]interface{}) (int, bool) {
	var maxNum int // 报警持续时间
	var count int  // 检查未通过的个数
	for i := 0; i < len(r.Rules); i++ {
		if r.Rules[i].IsTransferDurationRule() {
			// 循环区间流量报警
			if maxNum < 1 {
				maxNum = 1
			}
			for j := len(points[i]) - 1; j >= 0; j-- {
				if points[i][j] != nil {
					count++
					break
				}
			}
		} else {
			// 常规报警
			total := 0.0
			fail := 0.0
			num := int(r.Rules[i].Duration)
			if num > maxNum {
				maxNum = num
			}
			if len(points) < num {
				continue
			}
			for j := len(points) - 1; j >= 0 && len(points)-num <= j; j-- {
				total++
				if points[j][i] != nil {
					fail++
				}
			}
			// 当70%以上的采样点未通过规则判断时 才认为当前检查未通过
			if fail/total > 0.7 {
				count++
				break
			}
		}
	}
	// 仅当所有检查均未通过时 返回false
	return maxNum, count != len(r.Rules)
}
