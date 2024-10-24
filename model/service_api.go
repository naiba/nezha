package model

import "time"

type ServiceForm struct {
	Name                string          `json:"name,omitempty"`
	Target              string          `json:"target,omitempty"`
	Type                uint8           `json:"type,omitempty"`
	Cover               uint8           `json:"cover,omitempty"`
	Notify              bool            `json:"notify,omitempty"`
	Duration            uint64          `json:"duration,omitempty"`
	MinLatency          float32         `json:"min_latency,omitempty"`
	MaxLatency          float32         `json:"max_latency,omitempty"`
	LatencyNotify       bool            `json:"latency_notify,omitempty"`
	EnableTriggerTask   bool            `json:"enable_trigger_task,omitempty"`
	EnableShowInService bool            `json:"enable_show_in_service,omitempty"`
	FailTriggerTasks    []uint64        `json:"fail_trigger_tasks,omitempty"`
	RecoverTriggerTasks []uint64        `json:"recover_trigger_tasks,omitempty"`
	SkipServers         map[uint64]bool `json:"skip_servers,omitempty"`
	NotificationGroupID uint64          `json:"notification_group_id,omitempty"`
}

type ServiceResponseItem struct {
	Service     *Service
	CurrentUp   uint64
	CurrentDown uint64
	TotalUp     uint64
	TotalDown   uint64
	Delay       *[30]float32
	Up          *[30]int
	Down        *[30]int
}

func (r ServiceResponseItem) TotalUptime() float32 {
	if r.TotalUp+r.TotalDown == 0 {
		return 0
	}
	return float32(r.TotalUp) / (float32(r.TotalUp + r.TotalDown)) * 100
}

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

type ServiceResponse struct {
	Services           map[uint64]ServiceResponseItem
	CycleTransferStats map[uint64]CycleTransferStats
}
