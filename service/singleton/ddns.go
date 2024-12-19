package singleton

import (
	"cmp"
	"fmt"
	"slices"
	"sync"

	"github.com/libdns/cloudflare"
	tencentcloud "github.com/nezhahq/libdns-tencentcloud"

	"github.com/nezhahq/nezha/model"
	ddns2 "github.com/nezhahq/nezha/pkg/ddns"
	"github.com/nezhahq/nezha/pkg/ddns/dummy"
	"github.com/nezhahq/nezha/pkg/ddns/webhook"
	"github.com/nezhahq/nezha/pkg/utils"
)

var (
	DDNSCache     map[uint64]*model.DDNSProfile
	DDNSCacheLock sync.RWMutex
	DDNSList      []*model.DDNSProfile
	DDNSListLock  sync.RWMutex
)

func initDDNS() {
	DB.Find(&DDNSList)
	DDNSCache = make(map[uint64]*model.DDNSProfile)
	for i := 0; i < len(DDNSList); i++ {
		DDNSCache[DDNSList[i].ID] = DDNSList[i]
	}

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

	DDNSListLock.Lock()
	defer DDNSListLock.Unlock()

	DDNSList = utils.MapValuesToSlice(DDNSCache)
	slices.SortFunc(DDNSList, func(a, b *model.DDNSProfile) int {
		return cmp.Compare(a.ID, b.ID)
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
			DDNSCacheLock.RUnlock()
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
