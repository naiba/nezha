package dao

import (
	"sort"
	"sync"

	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"

	"github.com/naiba/nezha/model"
	pb "github.com/naiba/nezha/proto"
)

const (
	SnapshotDelay = 3
	ReportDelay   = 2
)

// Conf ..
var Conf *model.Config

// Cache ..
var Cache *cache.Cache

// DB ..
var DB *gorm.DB

// ServerList ..
var ServerList map[uint64]*model.Server
var SortedServerList []*model.Server

// ServerLock ..
var ServerLock sync.RWMutex

// Version ..
var Version = "debug"

func init() {
	if len(Version) > 7 {
		Version = Version[:7]
	}
}

func ReSortServer() {
	SortedServerList = []*model.Server{}
	for _, s := range ServerList {
		SortedServerList = append(SortedServerList, s)
	}

	sort.SliceStable(SortedServerList, func(i, j int) bool {
		return SortedServerList[i].DisplayIndex > SortedServerList[j].DisplayIndex
	})
}

// SendCommand ..
func SendCommand(cmd *pb.Command) {
	ServerLock.RLock()
	defer ServerLock.RUnlock()
	var err error
	for _, server := range ServerList {
		if server.Stream != nil {
			err = server.Stream.Send(cmd)
			if err != nil {
				close(server.StreamClose)
				server.Stream = nil
			}
		}
	}
}
