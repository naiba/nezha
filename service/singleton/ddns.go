package singleton

import (
	"fmt"
	"slices"
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
	DDNSList      []*model.DDNSProfile
)

func initDDNS() {
	var ddns []*model.DDNSProfile
	DB.Find(&ddns)
	DDNSCacheLock.Lock()
	DDNSCache = make(map[uint64]*model.DDNSProfile)
	for i := 0; i < len(ddns); i++ {
		DDNSCache[ddns[i].ID] = ddns[i]
	}
	DDNSCacheLock.Unlock()

	UpdateDDNSList()
	OnNameserverUpdate()
}

func OnDDNSUpdate(p *model.DDNSProfile) {
	DDNSCacheLock.Lock()
	defer DDNSCacheLock.Unlock()
	DDNSCache[p.ID] = p
}

func OnDDNSDelete(id []uint64) {
	DDNSCacheLock.Lock()
	defer DDNSCacheLock.Unlock()

	for _, i := range id {
		delete(DDNSCache, i)
	}
}

func UpdateDDNSList() {
	DDNSCacheLock.RLock()
	defer DDNSCacheLock.RUnlock()

	DDNSList = make([]*model.DDNSProfile, 0, len(DDNSCache))
	for _, p := range DDNSCache {
		DDNSList = append(DDNSList, p)
	}
	slices.SortFunc(DDNSList, func(a, b *model.DDNSProfile) int {
		if a.ID < b.ID {
			return -1
		} else if a.ID == b.ID {
			return 0
		}
		return 1
	})
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
