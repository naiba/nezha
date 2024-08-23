package singleton

import (
	"fmt"
	"log"
	"slices"

	ddns2 "github.com/naiba/nezha/pkg/ddns"
)

const (
	ProviderWebHook      = "webhook"
	ProviderCloudflare   = "cloudflare"
	ProviderTencentCloud = "tencentcloud"
)

type ProviderFunc func(*ddns2.DomainConfig) ddns2.Provider

func RetryableUpdateDomain(provider ddns2.Provider, domainConfig *ddns2.DomainConfig, maxRetries int) {
	if domainConfig == nil {
		return
	}
	for retries := 0; retries < maxRetries; retries++ {
		log.Printf("NEZHA>> 正在尝试更新域名(%s)DDNS(%d/%d)", domainConfig.FullDomain, retries+1, maxRetries)
		if err := provider.UpdateDomain(domainConfig); err != nil {
			log.Printf("NEZHA>> 尝试更新域名(%s)DDNS失败: %v", domainConfig.FullDomain, err)
		} else {
			log.Printf("NEZHA>> 尝试更新域名(%s)DDNS成功", domainConfig.FullDomain)
			break
		}
	}
}

// Deprecated
func GetDDNSProviderFromString(provider string) (ddns2.Provider, error) {
	switch provider {
	case ProviderWebHook:
		return ddns2.NewProviderWebHook(Conf.DDNS.WebhookURL, Conf.DDNS.WebhookMethod, Conf.DDNS.WebhookRequestBody, Conf.DDNS.WebhookHeaders), nil
	case ProviderCloudflare:
		return ddns2.NewProviderCloudflare(Conf.DDNS.AccessSecret), nil
	case ProviderTencentCloud:
		return ddns2.NewProviderTencentCloud(Conf.DDNS.AccessID, Conf.DDNS.AccessSecret), nil
	default:
		return new(ddns2.ProviderDummy), fmt.Errorf("无法找到配置的DDNS提供者 %s", provider)
	}
}

func GetDDNSProviderFromProfile(profileName string) (ddns2.Provider, error) {
	profile, ok := Conf.DDNS.Profiles[profileName]
	if !ok {
		return new(ddns2.ProviderDummy), fmt.Errorf("未找到配置项 %s", profileName)
	}

	switch profile.Provider {
	case ProviderWebHook:
		return ddns2.NewProviderWebHook(profile.WebhookURL, profile.WebhookMethod, profile.WebhookRequestBody, profile.WebhookHeaders), nil
	case ProviderCloudflare:
		return ddns2.NewProviderCloudflare(profile.AccessSecret), nil
	case ProviderTencentCloud:
		return ddns2.NewProviderTencentCloud(profile.AccessID, profile.AccessSecret), nil
	default:
		return new(ddns2.ProviderDummy), fmt.Errorf("无法找到配置的DDNS提供者 %s", profile.Provider)
	}
}

func ValidateDDNSProvidersFromProfiles() error {
	validProviders := []string{ProviderWebHook, ProviderCloudflare, ProviderTencentCloud}
	for _, profile := range Conf.DDNS.Profiles {
		if ok := slices.Contains(validProviders, profile.Provider); !ok {
			return fmt.Errorf("无法找到配置的DDNS提供者%s", profile.Provider)
		}
	}
	return nil
}
