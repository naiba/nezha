package model

import (
	"encoding/json"

	"gorm.io/gorm"
)

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

func (r *AlertRule) Snapshot(server *Server) []interface{} {
	var point []interface{}
	for i := 0; i < len(r.Rules); i++ {
		point = append(point, r.Rules[i].Snapshot(server))
	}
	return point
}

func (r *AlertRule) Check(points [][]interface{}) (int, bool) {
	var max int
	var count int
	for i := 0; i < len(r.Rules); i++ {
		total := 0.0
		fail := 0.0
		num := int(r.Rules[i].Duration / 2) // SnapshotDelay
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
	if count == len(r.Rules) {
		return max, false
	}
	return max, true
}
