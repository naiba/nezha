package dao

import (
	"sync"

	"github.com/jinzhu/gorm"
	"github.com/patrickmn/go-cache"

	"github.com/p14yground/nezha/model"
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
