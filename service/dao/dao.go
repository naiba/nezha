package dao

import (
	"sort"
	"sync"

	"github.com/patrickmn/go-cache"
	"github.com/robfig/cron/v3"
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

// 服务器监控、状态相关
var ServerList map[uint64]*model.Server
var ServerLock sync.RWMutex

var SortedServerList []*model.Server
var SortedServerLock sync.RWMutex

// 计划任务相关
var CronLock sync.RWMutex
var Crons map[uint64]*model.Cron
var Cron *cron.Cron

var Version = "v0.3.6"

func ReSortServer() {
	ServerLock.RLock()
	defer ServerLock.RUnlock()
	SortedServerLock.Lock()
	defer SortedServerLock.Unlock()

	SortedServerList = []*model.Server{}
	for _, s := range ServerList {
		SortedServerList = append(SortedServerList, s)
	}

	sort.SliceStable(SortedServerList, func(i, j int) bool {
		if SortedServerList[i].DisplayIndex == SortedServerList[j].DisplayIndex {
			return SortedServerList[i].ID < SortedServerList[i].ID
		}
		return SortedServerList[i].DisplayIndex > SortedServerList[j].DisplayIndex
	})
}
