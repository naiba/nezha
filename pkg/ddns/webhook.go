package ddns

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"github.com/naiba/nezha/pkg/utils"
)

type ProviderWebHook struct {
	url           string
	requestMethod string
	requestBody   string
	requestHeader string
	domainConfig  *DomainConfig
}

func NewProviderWebHook(s, rm, rb, rh string) *ProviderWebHook {
	return &ProviderWebHook{
		url:           s,
		requestMethod: rm,
		requestBody:   rb,
		requestHeader: rh,
	}
}

func (provider *ProviderWebHook) UpdateDomain(domainConfig *DomainConfig) error {
	if domainConfig == nil {
		return fmt.Errorf("获取 DDNS 配置失败")
	}
	provider.domainConfig = domainConfig

	if provider.domainConfig.FullDomain == "" {
		return fmt.Errorf("failed to update an empty domain")
	}

	if provider.domainConfig.EnableIPv4 && provider.domainConfig.Ipv4Addr != "" {
		url := provider.formatWebhookString(provider.url, "ipv4")
		body := provider.formatWebhookString(provider.requestBody, "ipv4")
		header := provider.formatWebhookString(provider.requestHeader, "ipv4")
		headers := strings.Split(header, "\n")
		req, err := http.NewRequest(provider.requestMethod, url, bytes.NewBufferString(body))
		if err == nil && req != nil {
			utils.SetStringHeadersToRequest(req, headers)
			if _, err := utils.HttpClient.Do(req); err != nil {
				return fmt.Errorf("failed to update a domain: %s. Cause by: %v", provider.domainConfig.FullDomain, err)
			}
		}
	}
	if provider.domainConfig.EnableIpv6 && provider.domainConfig.Ipv6Addr != "" {
		url := provider.formatWebhookString(provider.url, "ipv6")
		body := provider.formatWebhookString(provider.requestBody, "ipv6")
		header := provider.formatWebhookString(provider.requestHeader, "ipv6")
		headers := strings.Split(header, "\n")
		req, err := http.NewRequest(provider.requestMethod, url, bytes.NewBufferString(body))
		if err == nil && req != nil {
			utils.SetStringHeadersToRequest(req, headers)
			if _, err := utils.HttpClient.Do(req); err != nil {
				return fmt.Errorf("failed to update a domain: %s. Cause by: %v", provider.domainConfig.FullDomain, err)
			}
		}
	}
	return nil
}

func (provider *ProviderWebHook) formatWebhookString(s string, ipType string) string {
	if provider.domainConfig == nil {
		return s
	}

	result := strings.TrimSpace(s)
	result = strings.Replace(result, "{ip}", provider.domainConfig.Ipv4Addr, -1)
	result = strings.Replace(result, "{domain}", provider.domainConfig.FullDomain, -1)
	result = strings.Replace(result, "{type}", ipType, -1)
	// remove \r
	result = strings.Replace(result, "\r", "", -1)
	return result
}
