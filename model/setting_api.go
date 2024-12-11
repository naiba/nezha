package model

type SettingForm struct {
	DNSServers                  string `json:"dns_servers,omitempty" validate:"optional"`
	IgnoredIPNotification       string `json:"ignored_ip_notification,omitempty" validate:"optional"`
	IPChangeNotificationGroupID uint64 `json:"ip_change_notification_group_id,omitempty"` // IP变更提醒的通知组
	Cover                       uint8  `json:"cover,omitempty"`
	SiteName                    string `json:"site_name,omitempty" minLength:"1"`
	Language                    string `json:"language,omitempty" minLength:"2"`
	InstallHost                 string `json:"install_host,omitempty" validate:"optional"`
	CustomCode                  string `json:"custom_code,omitempty" validate:"optional"`
	CustomCodeDashboard         string `json:"custom_code_dashboard,omitempty" validate:"optional"`
	RealIPHeader                string `json:"real_ip_header,omitempty" validate:"optional"` // 真实IP
	UserTemplate                string `json:"user_template,omitempty" validate:"optional"`

	TLS                         bool `json:"tls,omitempty" validate:"optional"`
	EnableIPChangeNotification  bool `json:"enable_ip_change_notification,omitempty" validate:"optional"`
	EnablePlainIPInNotification bool `json:"enable_plain_ip_in_notification,omitempty" validate:"optional"`
}

type FrontendTemplate struct {
	Path       string `json:"path,omitempty"`
	Name       string `json:"name,omitempty"`
	Repository string `json:"repository,omitempty"`
	Author     string `json:"author,omitempty"`
	Version    string `json:"version,omitempty"`
	IsAdmin    bool   `json:"is_admin,omitempty"`
	IsOfficial bool   `json:"is_official,omitempty"`
}

type SettingResponse struct {
	Config

	Version           string             `json:"version,omitempty"`
	FrontendTemplates []FrontendTemplate `json:"frontend_templates,omitempty"`
}
