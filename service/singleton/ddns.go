package singleton

import (
	"fmt"
	"sync"

	"github.com/naiba/nezha/model"
	ddns2 "github.com/naiba/nezha/pkg/ddns"
)

var (
	ddnsCache     map[uint64]*model.DDNSProfile
	ddnsCacheLock sync.RWMutex
)

func initDDNS() {
	OnDDNSUpdate()
}

func OnDDNSUpdate() {
	var ddns []*model.DDNSProfile
	DB.Find(&ddns)
	ddnsCacheLock.Lock()
	defer ddnsCacheLock.Unlock()
	ddnsCache = make(map[uint64]*model.DDNSProfile)
	for i := 0; i < len(ddns); i++ {
		ddnsCache[ddns[i].ID] = ddns[i]
	}
}

func GetDDNSProvidersFromProfiles(profileId []uint64, ip *ddns2.IP) ([]ddns2.Provider, error) {
	profiles := make([]*model.DDNSProfile, 0, len(profileId))
	ddnsCacheLock.RLock()
	for _, id := range profileId {
		if profile, ok := ddnsCache[id]; ok {
			profiles = append(profiles, profile)
		} else {
			return nil, fmt.Errorf("无法找到DDNS配置 ID %d", id)
		}
	}
	ddnsCacheLock.RUnlock()

	providers := make([]ddns2.Provider, 0, len(profiles))
	for _, profile := range profiles {
		switch profile.Provider {
		case model.ProviderDummy:
			provider := ddns2.NewProviderDummy(profile)
			providers = append(providers, provider)
		case model.ProviderWebHook:
			provider := ddns2.NewProviderWebHook(profile, ip)
			providers = append(providers, provider)
		case model.ProviderCloudflare:
			provider := ddns2.NewProviderCloudflare(profile, ip)
			providers = append(providers, provider)
		case model.ProviderTencentCloud:
			provider := ddns2.NewProviderTencentCloud(profile, ip)
			providers = append(providers, provider)
		default:
			return nil, fmt.Errorf("无法找到配置的DDNS提供者ID %d", profile.Provider)
		}
	}
	return providers, nil
}
