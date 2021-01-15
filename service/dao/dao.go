package dao

import (
	"sort"
	"sync"

	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"

	"github.com/naiba/nezha/model"
)

const (
	SnapshotDelay = 3
	ReportDelay   = 2
)

var Conf *model.Config

var Cache *cache.Cache

var DB *gorm.DB

var ServerList map[uint64]*model.Server
var SortedServerList []*model.Server

var ServerLock sync.RWMutex

var Version = "v0.2.0"

func ReSortServer() {
	SortedServerList = []*model.Server{}
	for _, s := range ServerList {
		SortedServerList = append(SortedServerList, s)
	}

	sort.SliceStable(SortedServerList, func(i, j int) bool {
		return SortedServerList[i].DisplayIndex > SortedServerList[j].DisplayIndex
	})
}
