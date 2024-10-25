package model

type AlertRuleForm struct {
	ID                     uint64 `json:"id"`
	Name                   string `json:"name"`
	RulesRaw               string `json:"rules_raw"`
	FailTriggerTasksRaw    string `json:"fail_trigger_tasks_raw"`    // 失败时触发的任务id
	RecoverTriggerTasksRaw string `json:"recover_trigger_tasks_raw"` // 恢复时触发的任务id
	NotificationGroupID    uint64 `json:"notification_group_id"`
	TriggerMode            int    `json:"trigger_mode"`
	Enable                 bool   `json:"enable"`
}
