package singleton

import (
	"sort"
	"sync"

	"github.com/nezhahq/nezha/model"
)

var (
	ServerList     map[uint64]*model.Server // [ServerID] -> model.Server
	ServerUUIDToID map[string]uint64        // [ServerUUID] -> ServerID
	ServerLock     sync.RWMutex

	SortedServerList         []*model.Server // 用于存储服务器列表的 slice，按照服务器 ID 排序
	SortedServerListForGuest []*model.Server
	SortedServerLock         sync.RWMutex
)

func InitServer() {
	ServerList = make(map[uint64]*model.Server)
	ServerUUIDToID = make(map[string]uint64)
}

// loadServers 加载服务器列表并根据ID排序
func loadServers() {
	InitServer()
	var servers []model.Server
	DB.Find(&servers)
	for _, s := range servers {
		innerS := s
		innerS.Host = &model.Host{}
		innerS.State = &model.HostState{}
		innerS.GeoIP = new(model.GeoIP)
		ServerList[innerS.ID] = &innerS
		ServerUUIDToID[innerS.UUID] = innerS.ID
	}
	ReSortServer()
}

// ReSortServer 根据服务器ID 对服务器列表进行排序（ID越大越靠前）
func ReSortServer() {
	ServerLock.RLock()
	defer ServerLock.RUnlock()
	SortedServerLock.Lock()
	defer SortedServerLock.Unlock()

	SortedServerList = make([]*model.Server, 0, len(ServerList))
	SortedServerListForGuest = make([]*model.Server, 0)
	for _, s := range ServerList {
		SortedServerList = append(SortedServerList, s)
		if !s.HideForGuest {
			SortedServerListForGuest = append(SortedServerListForGuest, s)
		}
	}

	// 按照服务器 ID 排序的具体实现（ID越大越靠前）
	sort.SliceStable(SortedServerList, func(i, j int) bool {
		if SortedServerList[i].DisplayIndex == SortedServerList[j].DisplayIndex {
			return SortedServerList[i].ID < SortedServerList[j].ID
		}
		return SortedServerList[i].DisplayIndex > SortedServerList[j].DisplayIndex
	})

	sort.SliceStable(SortedServerListForGuest, func(i, j int) bool {
		if SortedServerListForGuest[i].DisplayIndex == SortedServerListForGuest[j].DisplayIndex {
			return SortedServerListForGuest[i].ID < SortedServerListForGuest[j].ID
		}
		return SortedServerListForGuest[i].DisplayIndex > SortedServerListForGuest[j].DisplayIndex
	})
}

func OnServerDelete(sid []uint64) {
	ServerLock.Lock()
	defer ServerLock.Unlock()
	for _, id := range sid {
		serverUUID := ServerList[id].UUID
		delete(ServerUUIDToID, serverUUID)
		delete(ServerList, id)
	}
}
