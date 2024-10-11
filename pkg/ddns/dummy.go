package ddns

import (
	"log"

	"github.com/naiba/nezha/model"
)

type ProviderDummy struct {
	provider
}

func NewProviderDummy(profile *model.DDNSProfile) *ProviderDummy {
	return &ProviderDummy{
		provider: provider{DDNSProfile: profile},
	}
}

func (provider *ProviderDummy) UpdateDomain() {
	for _, domain := range provider.DDNSProfile.Domains {
		for retries := 0; retries < int(provider.DDNSProfile.MaxRetries); retries++ {
			log.Printf("NEZHA>> 正在尝试更新域名(%s)DDNS(%d/%d)", domain, retries+1, provider.DDNSProfile.MaxRetries)
			log.Printf("NEZHA>> 尝试更新域名(%s)DDNS成功", domain)
		}
	}
}
