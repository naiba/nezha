package model

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
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
	Name     string
	RulesRaw string
	Enable   *bool
	Rules    []Rule `gorm:"-" json:"-"`
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
	return json.Unmarshal([]byte(r.RulesRaw), &r.Rules)
}

func (r *AlertRule) Enabled() bool {
	return r.Enable != nil && *r.Enable
}

func (r *AlertRule) Snapshot(cycleTransferStats *CycleTransferStats, server *Server, db *gorm.DB) []interface{} {
	var point []interface{}
	for i := 0; i < len(r.Rules); i++ {
		point = append(point, r.Rules[i].Snapshot(cycleTransferStats, server, db))
	}
	return point
}

func (r *AlertRule) Check(points [][]interface{}) (int, bool) {
	var max int
	var count int
	for i := 0; i < len(r.Rules); i++ {
		if r.Rules[i].IsTransferDurationRule() {
			// 循环区间流量报警
			if max < 1 {
				max = 1
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
			if num > max {
				max = num
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
			if fail/total > 0.7 {
				count++
				break
			}
		}
	}
	return max, count != len(r.Rules)
}
