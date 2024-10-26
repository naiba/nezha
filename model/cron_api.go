package model

type CronForm struct {
	ID                  uint64   `json:"id,omitempty"`
	TaskType            uint8    `json:"task_type,omitempty"` // 0:计划任务 1:触发任务
	Name                string   `json:"name,omitempty"`
	Scheduler           string   `json:"scheduler,omitempty"`
	Command             string   `json:"command,omitempty"`
	Servers             []uint64 `json:"servers,omitempty"`
	Cover               uint8    `json:"cover,omitempty"`
	PushSuccessful      bool     `json:"push_successful,omitempty"`
	NotificationGroupID uint64   `json:"notification_group_id,omitempty"`
}
