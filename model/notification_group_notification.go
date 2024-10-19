package model

type NotificationGroupNotification struct {
	Common
	NotificationGroupID uint64 `json:"notification_group_id"`
	NotificationID      uint64 `json:"notification_id"`
}
