package singleton

import (
	"sync"

	"github.com/naiba/nezha/model"
)

var natCache = make(map[string]*model.NAT)
var natCacheRwLock = new(sync.RWMutex)

func initNAT() {
	OnNATUpdate()
}

func OnNATUpdate() {
	natCacheRwLock.Lock()
	defer natCacheRwLock.Unlock()
	var nats []*model.NAT
	DB.Find(&nats)
	natCache = make(map[string]*model.NAT)
	for i := 0; i < len(nats); i++ {
		natCache[nats[i].Domain] = nats[i]
	}
}

func GetNATConfigByDomain(domain string) *model.NAT {
	natCacheRwLock.RLock()
	defer natCacheRwLock.RUnlock()
	return natCache[domain]
}
