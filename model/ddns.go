package model

import (
	"strings"

	"gorm.io/gorm"
)

const (
	ProviderDummy = iota
	ProviderWebHook
	ProviderCloudflare
	ProviderTencentCloud
)

const (
	_Dummy        = "dummy"
	_WebHook      = "webhook"
	_Cloudflare   = "cloudflare"
	_TencentCloud = "tencentcloud"
)

var ProviderMap = map[uint8]string{
	ProviderDummy:        _Dummy,
	ProviderWebHook:      _WebHook,
	ProviderCloudflare:   _Cloudflare,
	ProviderTencentCloud: _TencentCloud,
}

var ProviderList = []DDNSProvider{
	{
		Name: _Dummy,
		ID:   ProviderDummy,
	},
	{
		Name:         _Cloudflare,
		ID:           ProviderCloudflare,
		AccessSecret: true,
	},
	{
		Name:         _TencentCloud,
		ID:           ProviderTencentCloud,
		AccessID:     true,
		AccessSecret: true,
	},
	// Least frequently used, always place this at the end
	{
		Name:               _WebHook,
		ID:                 ProviderWebHook,
		AccessID:           true,
		AccessSecret:       true,
		WebhookURL:         true,
		WebhookMethod:      true,
		WebhookRequestType: true,
		WebhookRequestBody: true,
		WebhookHeaders:     true,
	},
}

type DDNSProfile struct {
	Common
	EnableIPv4         *bool
	EnableIPv6         *bool
	MaxRetries         uint64
	Name               string
	Provider           uint8
	AccessID           string
	AccessSecret       string
	WebhookURL         string
	WebhookMethod      uint8
	WebhookRequestType uint8
	WebhookRequestBody string
	WebhookHeaders     string

	Domains    []string `gorm:"-"`
	DomainsRaw string
}

func (d DDNSProfile) TableName() string {
	return "ddns"
}

func (d *DDNSProfile) AfterFind(tx *gorm.DB) error {
	if d.DomainsRaw != "" {
		d.Domains = strings.Split(d.DomainsRaw, ",")
	}
	return nil
}

type DDNSProvider struct {
	Name               string
	ID                 uint8
	AccessID           bool
	AccessSecret       bool
	WebhookURL         bool
	WebhookMethod      bool
	WebhookRequestType bool
	WebhookRequestBody bool
	WebhookHeaders     bool
}
