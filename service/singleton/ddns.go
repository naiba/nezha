package singleton

import (
	"fmt"
	"sync"

	"github.com/libdns/cloudflare"
	"github.com/libdns/tencentcloud"

	"github.com/naiba/nezha/model"
	ddns2 "github.com/naiba/nezha/pkg/ddns"
	"github.com/naiba/nezha/pkg/ddns/dummy"
	"github.com/naiba/nezha/pkg/ddns/webhook"
)

var (
	DDNSCache     map[uint64]*model.DDNSProfile
	DDNSCacheLock sync.RWMutex
)

func initDDNS() {
	OnDDNSUpdate()
	OnNameserverUpdate()
}

func OnDDNSUpdate() {
	var ddns []*model.DDNSProfile
	DB.Find(&ddns)
	DDNSCacheLock.Lock()
	defer DDNSCacheLock.Unlock()
	DDNSCache = make(map[uint64]*model.DDNSProfile)
	for i := 0; i < len(ddns); i++ {
		DDNSCache[ddns[i].ID] = ddns[i]
	}
}

func OnNameserverUpdate() {
	ddns2.InitDNSServers(Conf.DNSServers)
}

func GetDDNSProvidersFromProfiles(profileId []uint64, ip *ddns2.IP) ([]*ddns2.Provider, error) {
	profiles := make([]*model.DDNSProfile, 0, len(profileId))
	DDNSCacheLock.RLock()
	for _, id := range profileId {
		if profile, ok := DDNSCache[id]; ok {
			profiles = append(profiles, profile)
		} else {
			return nil, fmt.Errorf("无法找到DDNS配置 ID %d", id)
		}
	}
	DDNSCacheLock.RUnlock()

	providers := make([]*ddns2.Provider, 0, len(profiles))
	for _, profile := range profiles {
		provider := &ddns2.Provider{DDNSProfile: profile, IPAddrs: ip}
		switch profile.Provider {
		case model.ProviderDummy:
			provider.Setter = &dummy.Provider{}
			providers = append(providers, provider)
		case model.ProviderWebHook:
			provider.Setter = &webhook.Provider{DDNSProfile: profile}
			providers = append(providers, provider)
		case model.ProviderCloudflare:
			provider.Setter = &cloudflare.Provider{APIToken: profile.AccessSecret}
			providers = append(providers, provider)
		case model.ProviderTencentCloud:
			provider.Setter = &tencentcloud.Provider{SecretId: profile.AccessID, SecretKey: profile.AccessSecret}
			providers = append(providers, provider)
		default:
			return nil, fmt.Errorf("无法找到配置的DDNS提供者 %s", profile.Provider)
		}
	}
	return providers, nil
}
