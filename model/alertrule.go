package model

import (
	"encoding/json"

	"gorm.io/gorm"
)

type Rule struct {
	Type     string // 指标类型，cpu、memory、swap、disk、net_in、net_out、net_all、transfer_in、transfer_out、transfer_all、offline
	Min      uint64 // 最小阈值 (百分比、字节 kb ÷ 1024)
	Max      uint64 // 最大阈值 (百分比、字节 kb ÷ 1024)
	Duration uint64 // 持续时间 (秒)
}

type AlertRule struct {
	Common
	Name     string
	Rules    []Rule `gorm:"-" json:"-"`
	RulesRaw string
	Enable   *bool
}

func (r *AlertRule) BeforeSave(tx *gorm.DB) error {
	data, err := json.Marshal(r.Rules)
	if err != nil {
		return err
	}
	r.RulesRaw = string(data)
	return nil
}

func (r *AlertRule) AfterFind(tx *gorm.DB) error {
	return json.Unmarshal([]byte(r.RulesRaw), r.Rules)
}
