package dao

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/pkg/utils"
)

const (
	_RuleCheckNoData = iota
	_RuleCheckFail
	_RuleCheckPass
)

// 报警规则
var alertsLock sync.RWMutex
var alerts []*model.AlertRule
var alertsStore map[uint64]map[uint64][][]interface{}
var alertsPrevState map[uint64]map[uint64]uint

type NotificationHistory struct {
	Duration time.Duration
	Until    time.Time
}

func AlertSentinelStart() {
	alertsStore = make(map[uint64]map[uint64][][]interface{})
	alertsPrevState = make(map[uint64]map[uint64]uint)
	alertsLock.Lock()
	if err := DB.Find(&alerts).Error; err != nil {
		panic(err)
	}
	for i := 0; i < len(alerts); i++ {
		alertsStore[alerts[i].ID] = make(map[uint64][][]interface{})
		alertsPrevState[alerts[i].ID] = make(map[uint64]uint)
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
				log.Println("NEZHA>> 报警规则检测每小时", checkCount, "次", startedAt, time.Now())
			}
			checkCount = 0
			lastPrint = startedAt
		}
		time.Sleep(time.Until(startedAt.Add(time.Second * 3))) // 3秒钟检查一次
	}
}

func OnRefreshOrAddAlert(alert model.AlertRule) {
	alertsLock.Lock()
	defer alertsLock.Unlock()
	delete(alertsStore, alert.ID)
	delete(alertsPrevState, alert.ID)
	var isEdit bool
	for i := 0; i < len(alerts); i++ {
		if alerts[i].ID == alert.ID {
			alerts[i] = &alert
			isEdit = true
		}
	}
	if !isEdit {
		alerts = append(alerts, &alert)
	}
	alertsStore[alert.ID] = make(map[uint64][][]interface{})
	alertsPrevState[alert.ID] = make(map[uint64]uint)
}

func OnDeleteAlert(id uint64) {
	alertsLock.Lock()
	defer alertsLock.Unlock()
	delete(alertsStore, id)
	delete(alertsPrevState, id)
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
				ID][server.ID], alert.Snapshot(server, DB))
			// 发送通知，分为触发报警和恢复通知
			max, passed := alert.Check(alertsStore[alert.ID][server.ID])
			if !passed {
				alertsPrevState[alert.ID][server.ID] = _RuleCheckFail
				message := fmt.Sprintf("[主机故障] %s(%s) 规则：%s，", server.Name, utils.IPDesensitize(server.Host.IP), alert.Name)
				go SendNotification(message, true)
			} else {
				if alertsPrevState[alert.ID][server.ID] == _RuleCheckFail {
					message := fmt.Sprintf("[主机恢复] %s(%s) 规则：%s", server.Name, utils.IPDesensitize(server.Host.IP), alert.Name)
					go SendNotification(message, true)
				}
				alertsPrevState[alert.ID][server.ID] = _RuleCheckPass
			}
			// 清理旧数据
			if max > 0 && max < len(alertsStore[alert.ID][server.ID]) {
				alertsStore[alert.ID][server.ID] = alertsStore[alert.ID][server.ID][len(alertsStore[alert.ID][server.ID])-max:]
			}
		}
	}
}
