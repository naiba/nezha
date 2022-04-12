package singleton

import (
	"gorm.io/driver/sqlite"
	"time"

	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"

	"github.com/naiba/nezha/model"
)

var Version = "v0.12.18" // ！！记得修改 README 中的 badge 版本！！

var (
	Conf  *model.Config
	Cache *cache.Cache
	DB    *gorm.DB
	Loc   *time.Location
)

// Init 初始化singleton
func Init() {
	// 初始化时区至上海 UTF+8
	var err error
	Loc, err = time.LoadLocation("Asia/Shanghai")
	if err != nil {
		panic(err)
	}

	Conf = &model.Config{}
	Cache = cache.New(5*time.Minute, 10*time.Minute)
}

// LoadSingleton 加载子服务并执行
func LoadSingleton() {
	LoadNotifications() // 加载通知服务
	LoadServers()       // 加载服务器列表
	LoadCronTasks()     // 加载定时任务
}

// InitConfigFromPath 从给出的文件路径中加载配置
func InitConfigFromPath(path string) {
	err := Conf.Read(path)
	if err != nil {
		panic(err)
	}
}

// InitDBFromPath 从给出的文件路径中加载数据库
func InitDBFromPath(path string) {
	var err error
	DB, err = gorm.Open(sqlite.Open(path), &gorm.Config{
		CreateBatchSize: 200,
	})
	if err != nil {
		panic(err)
	}
	if Conf.Debug {
		DB = DB.Debug()
	}
	err = DB.AutoMigrate(model.Server{}, model.User{},
		model.Notification{}, model.AlertRule{}, model.Monitor{},
		model.MonitorHistory{}, model.Cron{}, model.Transfer{})
	if err != nil {
		panic(err)
	}
}
