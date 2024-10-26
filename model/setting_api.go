package model

type SettingForm struct {
	CustomNameservers           string `json:"custom_nameservers,omitempty"`
	IgnoredIPNotification       string `json:"ignored_ip_notification,omitempty"`
	IPChangeNotificationGroupID uint64 `json:"ip_change_notification_group_id,omitempty"` // IP变更提醒的通知组
	Cover                       uint8  `json:"cover,omitempty"`
	SiteName                    string `json:"site_name,omitempty"`

	EnableIPChangeNotification  bool `json:"enable_ip_change_notification,omitempty"`
	EnablePlainIPInNotification bool `json:"enable_plain_ip_in_notification,omitempty"`
}
