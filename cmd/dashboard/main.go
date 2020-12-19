package main

import (
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/patrickmn/go-cache"

	"github.com/naiba/nezha/cmd/dashboard/controller"
	"github.com/naiba/nezha/cmd/dashboard/rpc"
	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/service/dao"
)

func init() {
	var err error
	dao.ServerList = make(map[string]*model.Server)
	dao.Conf = &model.Config{}
	err = dao.Conf.Read("data/config.yaml")
	if err != nil {
		panic(err)
	}
	dao.DB, err = gorm.Open("sqlite3", "data/sqlite.db")
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
	dao.DB.AutoMigrate(model.Server{}, model.User{}, model.Notification{})
	// load cache
	var servers []model.Server
	dao.DB.Find(&servers)
	for _, s := range servers {
		innerS := s
		dao.ServerList[fmt.Sprintf("%d", innerS.ID)] = &innerS
	}
}

func main() {
	go controller.ServeWeb(dao.Conf.HTTPPort)
	go rpc.ServeRPC(5555)
	select {}
}
