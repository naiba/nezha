package singleton

import (
	"slices"
	"sync"

	"github.com/naiba/nezha/model"
)

var (
	NATCache       = make(map[string]*model.NAT)
	NATCacheRwLock sync.RWMutex

	NATIDToDomain = make(map[uint64]string)
	NATList       []*model.NAT
)

func initNAT() {
	DB.Find(&NATList)
	NATCacheRwLock.Lock()
	defer NATCacheRwLock.Unlock()
	NATCache = make(map[string]*model.NAT)
	for i := 0; i < len(NATList); i++ {
		NATCache[NATList[i].Domain] = NATList[i]
		NATIDToDomain[NATList[i].ID] = NATList[i].Domain
	}
}

func OnNATUpdate(n *model.NAT) {
	NATCacheRwLock.Lock()
	defer NATCacheRwLock.Unlock()

	if oldDomain, ok := NATIDToDomain[n.ID]; ok && oldDomain != n.Domain {
		delete(NATCache, oldDomain)
	}

	NATCache[n.Domain] = n
	NATIDToDomain[n.ID] = n.Domain
}

func OnNATDelete(id []uint64) {
	NATCacheRwLock.Lock()
	defer NATCacheRwLock.Unlock()

	for _, i := range id {
		if domain, ok := NATIDToDomain[i]; ok {
			delete(NATCache, domain)
			delete(NATIDToDomain, i)
		}
	}
}

func UpdateNATList() {
	NATCacheRwLock.RLock()
	defer NATCacheRwLock.RUnlock()

	NATList = make([]*model.NAT, 0, len(NATCache))
	for _, n := range NATCache {
		NATList = append(NATList, n)
	}
	slices.SortFunc(NATList, func(a, b *model.NAT) int {
		if a.ID < b.ID {
			return -1
		} else if a.ID == b.ID {
			return 0
		}
		return 1
	})
}

func GetNATConfigByDomain(domain string) *model.NAT {
	NATCacheRwLock.RLock()
	defer NATCacheRwLock.RUnlock()
	return NATCache[domain]
}
