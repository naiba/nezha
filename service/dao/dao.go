package dao

import (
	"fmt"
	"sort"
	"sync"

	"github.com/patrickmn/go-cache"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"

	"github.com/naiba/nezha/model"
	pb "github.com/naiba/nezha/proto"
)

var Version = "v0.7.1" // ！！记得修改 README 中的 badge 版本！！

const (
	SnapshotDelay = 3
	ReportDelay   = 2
)

var (
	Conf  *model.Config
	Cache *cache.Cache
	DB    *gorm.DB

	ServerList map[uint64]*model.Server
	SecretToID map[string]uint64
	ServerLock sync.RWMutex

	SortedServerList []*model.Server
	SortedServerLock sync.RWMutex
)

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
			return SortedServerList[i].ID < SortedServerList[j].ID
		}
		return SortedServerList[i].DisplayIndex > SortedServerList[j].DisplayIndex
	})
}

// =============== Cron Mixin ===============

var CronLock sync.RWMutex
var Crons map[uint64]*model.Cron
var Cron *cron.Cron

func CronTrigger(c *model.Cron) {
	ServerLock.RLock()
	defer ServerLock.RUnlock()
	for j := 0; j < len(c.Servers); j++ {
		if ServerList[c.Servers[j]].TaskStream != nil {
			ServerList[c.Servers[j]].TaskStream.Send(&pb.Task{
				Id:   c.ID,
				Data: c.Command,
				Type: model.TaskTypeCommand,
			})
		} else {
			SendNotification(fmt.Sprintf("计划任务：%s，服务器：%d 离线，无法执行。", c.Name, c.Servers[j]), false)
		}
	}
}
