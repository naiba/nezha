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
	go checkStatus()
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
		}
	}
}

func checkStatus() {
	startedAt := time.Now()
	defer func() {
		time.Sleep(time.Until(startedAt.Add(time.Second * dao.SnapshotDelay)))
		checkStatus()
	}()

	alertsLock.RLock()
	defer alertsLock.RUnlock()
	dao.ServerLock.RLock()
	defer dao.ServerLock.RUnlock()

	for j := 0; j < len(alerts); j++ {
		for _, server := range dao.ServerList {
			// 监测点
			alertsStore[alerts[j].ID][server.ID] = append(alertsStore[alerts[j].
				ID][server.ID], alerts[j].Snapshot(server))
			// 发送通知
			max, desc := alerts[j].Check(alertsStore[alerts[j].ID][server.ID])
			if desc != "" {
				nID := getNotificationHash(server, desc)
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
						dao.Cache.Set(nID, nHistory, nHistory.Duration)
					}
				} else {
					// 新提醒直接通知
					flag = true
					dao.Cache.Set(nID, NotificationHistory{
						Duration: firstNotificationDelay,
						Until:    time.Now().Add(firstNotificationDelay),
					}, firstNotificationDelay)
				}
				if flag {
					message := fmt.Sprintf("逮到咯，快去看看！服务器：%s(%s)，报警规则：%s，%s", server.Name, server.Host.IP, alerts[j].Name, desc)
					log.Printf("通知：%s\n", message)
					go sendNotification(message)
				}
			}
			// 清理旧数据
			if max > 0 && max < len(alertsStore[alerts[j].ID][server.ID]) {
				alertsStore[alerts[j].ID][server.ID] = alertsStore[alerts[j].ID][server.ID][len(alertsStore[alerts[j].ID][server.ID])-max:]
			}
		}
	}
}

func sendNotification(desc string) {
	notificationsLock.RLock()
	defer notificationsLock.RUnlock()
	for i := 0; i < len(notifications); i++ {
		notifications[i].Send(desc)
	}
}

func getNotificationHash(server *model.Server, desc string) string {
	return hex.EncodeToString(md5.New().Sum([]byte(fmt.Sprintf("%d::%s", server.ID, desc))))
}
