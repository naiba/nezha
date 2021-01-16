package alertmanager

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/service/dao"
)

const firstNotificationDelay = time.Minute * 15

// 通知方式
var notifications []model.Notification
var notificationsLock sync.RWMutex

// 报警规则
var alertsLock sync.RWMutex
var alerts []model.AlertRule
var alertsStore map[uint64]map[uint64][][]interface{}

type NotificationHistory struct {
	Duration time.Duration
	Until    time.Time
}

func Start() {
	alertsStore = make(map[uint64]map[uint64][][]interface{})
	notificationsLock.Lock()
	if err := dao.DB.Find(&notifications).Error; err != nil {
		panic(err)
	}
	notificationsLock.Unlock()
	alertsLock.Lock()
	if err := dao.DB.Find(&alerts).Error; err != nil {
		panic(err)
	}
	for i := 0; i < len(alerts); i++ {
		alertsStore[alerts[i].ID] = make(map[uint64][][]interface{})
	}
	alertsLock.Unlock()

	time.Sleep(time.Second * 10)
	var lastPrint time.Time
	var checkCount uint64
	for {
		startedAt := time.Now()
		checkStatus()
		checkCount++
		if lastPrint.Before(startedAt.Add(-1 * time.Hour)) {
			log.Println("报警规则检测每小时", checkCount, "次", startedAt, time.Now())
			checkCount = 0
			lastPrint = startedAt
		}
		time.Sleep(time.Until(startedAt.Add(time.Second * dao.SnapshotDelay)))
	}
}

func OnRefreshOrAddAlert(alert model.AlertRule) {
	alertsLock.Lock()
	defer alertsLock.Unlock()
	delete(alertsStore, alert.ID)
	var isEdit bool
	for i := 0; i < len(alerts); i++ {
		if alerts[i].ID == alert.ID {
			alerts[i] = alert
			isEdit = true
		}
	}
	if !isEdit {
		alerts = append(alerts, alert)
	}
	alertsStore[alert.ID] = make(map[uint64][][]interface{})
}

func OnDeleteAlert(id uint64) {
	alertsLock.Lock()
	defer alertsLock.Unlock()
	delete(alertsStore, id)
	for i := 0; i < len(alerts); i++ {
		if alerts[i].ID == id {
			alerts = append(alerts[:i], alerts[i+1:]...)
			i--
		}
	}
}

func OnRefreshOrAddNotification(n model.Notification) {
	notificationsLock.Lock()
	defer notificationsLock.Unlock()
	var isEdit bool
	for i := 0; i < len(notifications); i++ {
		if notifications[i].ID == n.ID {
			notifications[i] = n
			isEdit = true
		}
	}
	if !isEdit {
		notifications = append(notifications, n)
	}
}

func OnDeleteNotification(id uint64) {
	notificationsLock.Lock()
	defer notificationsLock.Unlock()
	for i := 0; i < len(notifications); i++ {
		if notifications[i].ID == id {
			notifications = append(notifications[:i], notifications[i+1:]...)
			i--
		}
	}
}

func checkStatus() {
	alertsLock.RLock()
	defer alertsLock.RUnlock()
	dao.ServerLock.RLock()
	defer dao.ServerLock.RUnlock()

	for _, alert := range alerts {
		// 跳过未启用
		if alert.Enable == nil || !*alert.Enable {
			continue
		}
		for _, server := range dao.ServerList {
			// 监测点
			alertsStore[alert.ID][server.ID] = append(alertsStore[alert.
				ID][server.ID], alert.Snapshot(server))
			// 发送通知
			max, desc := alert.Check(alertsStore[alert.ID][server.ID])
			if desc != "" {
				message := fmt.Sprintf("报警规则：%s，服务器：%s(%s)，%s，逮到咯，快去看看！", alert.Name, server.Name, server.Host.IP, desc)
				go SendNotification(message)
			}
			// 清理旧数据
			if max > 0 && max < len(alertsStore[alert.ID][server.ID]) {
				alertsStore[alert.ID][server.ID] = alertsStore[alert.ID][server.ID][len(alertsStore[alert.ID][server.ID])-max:]
			}
		}
	}
}

func SendNotification(desc string) {
	// 通知防骚扰策略
	nID := hex.EncodeToString(md5.New().Sum([]byte(desc)))
	var flag bool
	if cacheN, has := dao.Cache.Get(nID); has {
		nHistory := cacheN.(NotificationHistory)
		// 每次提醒都增加一倍等待时间，最后每天最多提醒一次
		if time.Now().After(nHistory.Until) {
			flag = true
			nHistory.Duration *= 2
			if nHistory.Duration > time.Hour*24 {
				nHistory.Duration = time.Hour * 24
			}
			nHistory.Until = time.Now().Add(nHistory.Duration)
			// 缓存有效期加 10 分钟
			dao.Cache.Set(nID, nHistory, nHistory.Duration+time.Minute*10)
		}
	} else {
		// 新提醒直接通知
		flag = true
		dao.Cache.Set(nID, NotificationHistory{
			Duration: firstNotificationDelay,
			Until:    time.Now().Add(firstNotificationDelay),
		}, firstNotificationDelay+time.Minute*10)
	}

	if !flag {
		return
	}

	// 发出通知
	notificationsLock.RLock()
	defer notificationsLock.RUnlock()
	for i := 0; i < len(notifications); i++ {
		notifications[i].Send(desc)
	}
}
