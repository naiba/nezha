package singleton

import (
	"sort"
	"sync"

	"github.com/naiba/nezha/model"
)

var (
	ServerList map[uint64]*model.Server // [ServerID] -> model.Server
	SecretToID map[string]uint64        // [ServerSecret] -> ServerID
	ServerLock sync.RWMutex

	SortedServerList []*model.Server // 用于存储服务器列表的 slice，按照服务器 ID 排序
	SortedServerLock sync.RWMutex
)

// InitServer 初始化 ServerID <-> Secret 的映射
func InitServer() {
	ServerList = make(map[uint64]*model.Server)
	SecretToID = make(map[string]uint64)
}

//LoadServers 加载服务器列表并根据ID排序
func LoadServers() {
	InitServer()
	var servers []model.Server
	DB.Find(&servers)
	for _, s := range servers {
		innerS := s
		innerS.Host = &model.Host{}
		innerS.State = &model.HostState{}
		ServerList[innerS.ID] = &innerS
		SecretToID[innerS.Secret] = innerS.ID
	}
	ReSortServer()
}

// ReSortServer 根据服务器ID 对服务器列表进行排序（ID越大越靠前）
func ReSortServer() {
	ServerLock.RLock()
	defer ServerLock.RUnlock()
	SortedServerLock.Lock()
	defer SortedServerLock.Unlock()

	SortedServerList = []*model.Server{}
	for _, s := range ServerList {
		SortedServerList = append(SortedServerList, s)
	}

	// 按照服务器 ID 排序的具体实现（ID越大越靠前）
	sort.SliceStable(SortedServerList, func(i, j int) bool {
		if SortedServerList[i].DisplayIndex == SortedServerList[j].DisplayIndex {
			return SortedServerList[i].ID < SortedServerList[j].ID
		}
		return SortedServerList[i].DisplayIndex > SortedServerList[j].DisplayIndex
	})
}
