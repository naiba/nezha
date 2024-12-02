package model

type AlertRuleForm struct {
	Name                string   `json:"name,omitempty" minLength:"1"`
	Rules               []Rule   `json:"rules,omitempty"`
	FailTriggerTasks    []uint64 `json:"fail_trigger_tasks,omitempty"`    // 失败时触发的任务id
	RecoverTriggerTasks []uint64 `json:"recover_trigger_tasks,omitempty"` // 恢复时触发的任务id
	NotificationGroupID uint64   `json:"notification_group_id,omitempty"`
	TriggerMode         uint8    `json:"trigger_mode,omitempty" default:"0"`
	Enable              bool     `json:"enable,omitempty" validate:"optional"`
}
