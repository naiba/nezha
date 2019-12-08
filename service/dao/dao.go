package dao

import (
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
