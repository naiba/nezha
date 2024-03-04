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
	}
	return ddns2.ProviderDummy{}, errors.New(fmt.Sprintf("无法找到配置的DDNS提供者%s", Conf.DDNS.Provider))
}
