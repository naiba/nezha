package main

import (
	"time"

	"github.com/patrickmn/go-cache"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/naiba/nezha/cmd/dashboard/controller"
	"github.com/naiba/nezha/cmd/dashboard/rpc"
	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/service/alertmanager"
	"github.com/naiba/nezha/service/dao"
)

func init() {
	var err error
	dao.ServerList = make(map[uint64]*model.Server)
	dao.Conf = &model.Config{}
	err = dao.Conf.Read("data/config.yaml")
	if err != nil {
		panic(err)
	}
	dao.DB, err = gorm.Open(sqlite.Open("data/sqlite.db"), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	if dao.Conf.Debug {
		dao.DB = dao.DB.Debug()
	}
	dao.Cache = cache.New(5*time.Minute, 10*time.Minute)
	initDB()
}

func initDB() {
	dao.DB.AutoMigrate(model.Server{}, model.User{}, model.Notification{}, model.AlertRule{})
	// load cache
	var servers []model.Server
	dao.DB.Find(&servers)
	for _, s := range servers {
		innerS := s
		innerS.Host = &model.Host{}
		innerS.State = &model.State{}
		dao.ServerList[innerS.ID] = &innerS
	}
	dao.ReSortServer()
}

func main() {
	go controller.ServeWeb(dao.Conf.HTTPPort)
	go rpc.ServeRPC(5555)
	alertmanager.Start()
}
