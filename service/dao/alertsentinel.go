package dao

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/naiba/nezha/model"
)

// 报警规则
var alertsLock sync.RWMutex
var alerts []model.AlertRule
var alertsStore map[uint64]map[uint64][][]interface{}

type NotificationHistory struct {
	Duration time.Duration
	Until    time.Time
}

func AlertSentinelStart() {
	alertsStore = make(map[uint64]map[uint64][][]interface{})
	notificationsLock.Lock()
	if err := DB.Find(&notifications).Error; err != nil {
		panic(err)
	}
	notificationsLock.Unlock()
	alertsLock.Lock()
	if err := DB.Find(&alerts).Error; err != nil {
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
			if Conf.Debug {
				log.Println("报警规则检测每小时", checkCount, "次", startedAt, time.Now())
			}
			checkCount = 0
			lastPrint = startedAt
		}
		time.Sleep(time.Until(startedAt.Add(time.Second * SnapshotDelay)))
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

func checkStatus() {
	alertsLock.RLock()
	defer alertsLock.RUnlock()
	ServerLock.RLock()
	defer ServerLock.RUnlock()

	for _, alert := range alerts {
		// 跳过未启用
		if alert.Enable == nil || !*alert.Enable {
			continue
		}
		for _, server := range ServerList {
			// 监测点
			alertsStore[alert.ID][server.ID] = append(alertsStore[alert.
				ID][server.ID], alert.Snapshot(server))
			// 发送通知
			max, desc := alert.Check(alertsStore[alert.ID][server.ID])
			if desc != "" {
				message := fmt.Sprintf("报警规则：%s，服务器：%s(%s)，%s，逮到咯，快去看看！", alert.Name, server.Name, server.Host.IP, desc)
				go SendNotification(message, true)
			}
			// 清理旧数据
			if max > 0 && max < len(alertsStore[alert.ID][server.ID]) {
				alertsStore[alert.ID][server.ID] = alertsStore[alert.ID][server.ID][len(alertsStore[alert.ID][server.ID])-max:]
			}
		}
	}
}
