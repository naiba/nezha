package dao

import (
	"sync"

	"github.com/jinzhu/gorm"
	"github.com/patrickmn/go-cache"

	"github.com/p14yground/nezha/model"
	pb "github.com/p14yground/nezha/proto"
)

// Conf ..
var Conf *model.Config

// Cache ..
var Cache *cache.Cache

// DB ..
var DB *gorm.DB

// Admin ..
var Admin *model.User

// ServerList ..
var ServerList map[string]*model.Server

// ServerLock ..
var ServerLock sync.RWMutex

// Version ..
var Version = "debug"

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
