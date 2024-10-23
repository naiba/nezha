package model

type MonitorForm struct {
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
