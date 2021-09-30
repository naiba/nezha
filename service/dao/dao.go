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

var Version = "v0.10.4" // ！！记得修改 README 中的 badge 版本！！

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

func ManualTrigger(c *model.Cron) {
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
			SendNotification(fmt.Sprintf("[任务失败] %s，服务器 %s 离线，无法执行。", c.Name, ServerList[c.Servers[j]].Name), false)
		}
	}
}

func CronTrigger(cr model.Cron) func() {
	crIgnoreMap := make(map[uint64]bool)
	for j := 0; j < len(cr.Servers); j++ {
		crIgnoreMap[cr.Servers[j]] = true
	}
	return func() {
		ServerLock.RLock()
		defer ServerLock.RUnlock()
		for _, s := range ServerList {
			if cr.Cover == model.CronCoverAll && crIgnoreMap[s.ID] {
				continue
			}
			if cr.Cover == model.CronCoverIgnoreAll && !crIgnoreMap[s.ID] {
				continue
			}
			if s.TaskStream != nil {
				s.TaskStream.Send(&pb.Task{
					Id:   cr.ID,
					Data: cr.Command,
					Type: model.TaskTypeCommand,
				})
			} else {
				SendNotification(fmt.Sprintf("[任务失败] %s，服务器 %s 离线，无法执行。", cr.Name, s.Name), false)
			}
		}
	}
}
