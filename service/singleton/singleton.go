package singleton

import (
	_ "embed"
	"log"
	"time"

	"github.com/patrickmn/go-cache"
	"gopkg.in/yaml.v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/nezhahq/nezha/model"
	"github.com/nezhahq/nezha/pkg/utils"
)

var Version = "debug"

var (
	Conf              *model.Config
	Cache             *cache.Cache
	DB                *gorm.DB
	Loc               *time.Location
	FrontendTemplates []model.FrontendTemplate
	DashboardBootTime = uint64(time.Now().Unix())
)

//go:embed frontend-templates.yaml
var frontendTemplatesYAML []byte

func InitTimezoneAndCache() {
	var err error
	Loc, err = time.LoadLocation(Conf.Location)
	if err != nil {
		panic(err)
	}

	Cache = cache.New(5*time.Minute, 10*time.Minute)
}

// LoadSingleton 加载子服务并执行
func LoadSingleton() {
	initI18n()          // 加载本地化服务
	loadNotifications() // 加载通知服务
	loadServers()       // 加载服务器列表
	loadCronTasks()     // 加载定时任务
	initNAT()
	initDDNS()
}

// InitFrontendTemplates 从内置文件中加载FrontendTemplates
func InitFrontendTemplates() {
	err := yaml.Unmarshal(frontendTemplatesYAML, &FrontendTemplates)
	if err != nil {
		panic(err)
	}
}

// InitConfigFromPath 从给出的文件路径中加载配置
func InitConfigFromPath(path string) {
	Conf = &model.Config{}
	err := Conf.Read(path, FrontendTemplates)
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
	err = DB.AutoMigrate(model.Server{}, model.User{}, model.ServerGroup{}, model.NotificationGroup{},
		model.Notification{}, model.AlertRule{}, model.Service{}, model.NotificationGroupNotification{},
		model.ServiceHistory{}, model.Cron{}, model.Transfer{}, model.ServerGroupServer{},
		model.NAT{}, model.DDNSProfile{}, model.NotificationGroupNotification{},
		model.WAF{})
	if err != nil {
		panic(err)
	}
}

// RecordTransferHourlyUsage 对流量记录进行打点
func RecordTransferHourlyUsage() {
	ServerLock.Lock()
	defer ServerLock.Unlock()
	now := time.Now()
	nowTrimSeconds := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	var txs []model.Transfer
	for id, server := range ServerList {
		tx := model.Transfer{
			ServerID: id,
			In:       utils.Uint64SubInt64(server.State.NetInTransfer, server.PrevTransferInSnapshot),
			Out:      utils.Uint64SubInt64(server.State.NetOutTransfer, server.PrevTransferOutSnapshot),
		}
		if tx.In == 0 && tx.Out == 0 {
			continue
		}
		server.PrevTransferInSnapshot = int64(server.State.NetInTransfer)
		server.PrevTransferOutSnapshot = int64(server.State.NetOutTransfer)
		tx.CreatedAt = nowTrimSeconds
		txs = append(txs, tx)
	}
	if len(txs) == 0 {
		return
	}
	log.Println("NEZHA>> Cron 流量统计入库", len(txs), DB.Create(txs).Error)
}

// CleanServiceHistory 清理无效或过时的 监控记录 和 流量记录
func CleanServiceHistory() {
	// 清理已被删除的服务器的监控记录与流量记录
	DB.Unscoped().Delete(&model.ServiceHistory{}, "created_at < ? OR service_id NOT IN (SELECT `id` FROM services)", time.Now().AddDate(0, 0, -30))
	// 由于网络监控记录的数据较多，并且前端仅使用了 1 天的数据
	// 考虑到 sqlite 数据量问题，仅保留一天数据，
	// server_id = 0 的数据会用于/service页面的可用性展示
	DB.Unscoped().Delete(&model.ServiceHistory{}, "(created_at < ? AND server_id != 0) OR service_id NOT IN (SELECT `id` FROM services)", time.Now().AddDate(0, 0, -1))
	DB.Unscoped().Delete(&model.Transfer{}, "server_id NOT IN (SELECT `id` FROM servers)")
	// 计算可清理流量记录的时长
	var allServerKeep time.Time
	specialServerKeep := make(map[uint64]time.Time)
	var specialServerIDs []uint64
	var alerts []model.AlertRule
	DB.Find(&alerts)
	for _, alert := range alerts {
		for _, rule := range alert.Rules {
			// 是不是流量记录规则
			if !rule.IsTransferDurationRule() {
				continue
			}
			dataCouldRemoveBefore := rule.GetTransferDurationStart().UTC()
			// 判断规则影响的机器范围
			if rule.Cover == model.RuleCoverAll {
				// 更新全局可以清理的数据点
				if allServerKeep.IsZero() || allServerKeep.After(dataCouldRemoveBefore) {
					allServerKeep = dataCouldRemoveBefore
				}
			} else {
				// 更新特定机器可以清理数据点
				for id := range rule.Ignore {
					if specialServerKeep[id].IsZero() || specialServerKeep[id].After(dataCouldRemoveBefore) {
						specialServerKeep[id] = dataCouldRemoveBefore
						specialServerIDs = append(specialServerIDs, id)
					}
				}
			}
		}
	}
	for id, couldRemove := range specialServerKeep {
		DB.Unscoped().Delete(&model.Transfer{}, "server_id = ? AND datetime(`created_at`) < datetime(?)", id, couldRemove)
	}
	if allServerKeep.IsZero() {
		DB.Unscoped().Delete(&model.Transfer{}, "server_id NOT IN (?)", specialServerIDs)
	} else {
		DB.Unscoped().Delete(&model.Transfer{}, "server_id NOT IN (?) AND datetime(`created_at`) < datetime(?)", specialServerIDs, allServerKeep)
	}
}

// IPDesensitize 根据设置选择是否对IP进行打码处理 返回处理后的IP(关闭打码则返回原IP)
func IPDesensitize(ip string) string {
	if Conf.EnablePlainIPInNotification {
		return ip
	}
	return utils.IPDesensitize(ip)
}
