package model

type AlertRuleForm struct {
	ID                  uint64   `json:"id"`
	Name                string   `json:"name"`
	Rules               []Rule   `json:"rules"`
	FailTriggerTasks    []uint64 `json:"fail_trigger_tasks"`    // 失败时触发的任务id
	RecoverTriggerTasks []uint64 `json:"recover_trigger_tasks"` // 恢复时触发的任务id
	NotificationGroupID uint64   `json:"notification_group_id"`
	TriggerMode         int      `json:"trigger_mode"`
	Enable              bool     `json:"enable"`
}
