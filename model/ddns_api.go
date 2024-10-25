package model

type DDNSForm struct {
	ID                 uint64   `json:"id,omitempty"`
	MaxRetries         uint64   `json:"max_retries,omitempty"`
	EnableIPv4         bool     `json:"enable_ipv4,omitempty"`
	EnableIPv6         bool     `json:"enable_ipv6,omitempty"`
	Name               string   `json:"name,omitempty"`
	Provider           string   `json:"provider,omitempty"`
	Domains            []string `json:"domains,omitempty"`
	AccessID           string   `json:"access_id,omitempty"`
	AccessSecret       string   `json:"access_secret,omitempty"`
	WebhookURL         string   `json:"webhook_url,omitempty"`
	WebhookMethod      uint8    `json:"webhook_method,omitempty"`
	WebhookRequestType uint8    `json:"webhook_request_type,omitempty"`
	WebhookRequestBody string   `json:"webhook_request_body,omitempty"`
	WebhookHeaders     string   `json:"webhook_headers,omitempty"`
}
