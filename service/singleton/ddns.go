package singleton

import (
	"errors"
	"fmt"
	ddns2 "github.com/naiba/nezha/pkg/ddns"
	"log"
)

func RetryableUpdateDomain(provider ddns2.Provider, config *ddns2.DomainConfig, maxRetries int) bool {
	if nil == config {
		return false
	}
	for retries := 0; retries < maxRetries; retries++ {
		log.Printf("NEZHA>> 正在尝试更新域名(%s)DDNS(%d/%d)\n", config.FullDomain, retries+1, maxRetries)
		if provider.UpdateDomain(config) {
			log.Printf("NEZHA>> 尝试更新域名(%s)DDNS成功\n", config.FullDomain)
			return true
		}
	}
	log.Printf("NEZHA>> 尝试更新域名(%s)DDNS失败\n", config.FullDomain)
	return false
}

func GetDDNSProviderFromString(provider string) (ddns2.Provider, error) {
	switch provider {
	case "webhook":
		return ddns2.ProviderWebHook{
			URL:           Conf.DDNS.WebhookURL,
			RequestMethod: Conf.DDNS.WebhookMethod,
			RequestBody:   Conf.DDNS.WebhookRequestBody,
			RequestHeader: Conf.DDNS.WebhookHeaders,
		}, nil
	case "dummy":
		return ddns2.ProviderDummy{}, nil
	case "cloudflare":
		return ddns2.ProviderCloudflare{
			Secret: Conf.DDNS.AccessSecret,
		}, nil
	case "tencentcloud":
		return ddns2.ProviderTencentCloud{
			SecretID:  Conf.DDNS.AccessID,
			SecretKey: Conf.DDNS.AccessSecret,
		}, nil
	}
	return ddns2.ProviderDummy{}, errors.New(fmt.Sprintf("无法找到配置的DDNS提供者%s", Conf.DDNS.Provider))
}

func GetDDNSProviderFromProfile(profileName string) (ddns2.Provider, error) {
	profile, ok := Conf.DDNS.Profiles[profileName]
	if !ok {
		return ddns2.ProviderDummy{}, errors.New(fmt.Sprintf("未找到配置项 %s", profileName))
	}

	switch profile.Provider {
	case "webhook":
		return ddns2.ProviderWebHook{
			URL:           profile.WebhookURL,
			RequestMethod: profile.WebhookMethod,
			RequestBody:   profile.WebhookRequestBody,
			RequestHeader: profile.WebhookHeaders,
		}, nil
	case "dummy":
		return ddns2.ProviderDummy{}, nil
	case "cloudflare":
		return ddns2.ProviderCloudflare{
			Secret: profile.AccessSecret,
		}, nil
	case "tencentcloud":
		return ddns2.ProviderTencentCloud{
			SecretID:  profile.AccessID,
			SecretKey: profile.AccessSecret,
		}, nil
	}
	return ddns2.ProviderDummy{}, errors.New(fmt.Sprintf("无法找到配置的DDNS提供者%s", profile.Provider))
}

func ValidateDDNSProvidersFromProfiles() error {
	validProviders := map[string]bool{"webhook": true, "dummy": true, "cloudflare": true, "tencentcloud": true}
	providers := make(map[string]string)
	for profileName, profile := range Conf.DDNS.Profiles {
		if _, ok := validProviders[profile.Provider]; !ok {
			return errors.New(fmt.Sprintf("无法找到配置的DDNS提供者%s", profile.Provider))
		}
		providers[profileName] = profile.Provider
	}
	return nil
}
