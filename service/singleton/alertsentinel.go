package singleton

import (
	"fmt"
	"github.com/jinzhu/copier"
	"log"
	"sync"
	"time"

	"github.com/naiba/nezha/model"
)

const (
	_RuleCheckNoData = iota
	_RuleCheckFail
	_RuleCheckPass
)

type NotificationHistory struct {
	Duration time.Duration
	Until    time.Time
}

// 报警规则
var (
	AlertsLock                    sync.RWMutex
	Alerts                        []*model.AlertRule
	alertsStore                   map[uint64]map[uint64][][]interface{} // [alert_id][server_id] -> 对应报警规则的检查结果
	alertsPrevState               map[uint64]map[uint64]uint            // [alert_id][server_id] -> 对应报警规则的上一次报警状态
	AlertsCycleTransferStatsStore map[uint64]*model.CycleTransferStats  // [alert_id] -> 对应报警规则的周期流量统计
)

// addCycleTransferStatsInfo 向AlertsCycleTransferStatsStore中添加周期流量报警统计信息
func addCycleTransferStatsInfo(alert *model.AlertRule) {
	if !alert.Enabled() {
		return
	}
	for j := 0; j < len(alert.Rules); j++ {
		if !alert.Rules[j].IsTransferDurationRule() {
			continue
		}
		if AlertsCycleTransferStatsStore[alert.ID] == nil {
			from := alert.Rules[j].GetTransferDurationStart()
			to := alert.Rules[j].GetTransferDurationEnd()
			AlertsCycleTransferStatsStore[alert.ID] = &model.CycleTransferStats{
				Name:       alert.Name,
				From:       from,
				To:         to,
				Max:        uint64(alert.Rules[j].Max),
				Min:        uint64(alert.Rules[j].Min),
				ServerName: make(map[uint64]string),
				Transfer:   make(map[uint64]uint64),
				NextUpdate: make(map[uint64]time.Time),
			}
		}
	}
}

// AlertSentinelStart 报警器启动
func AlertSentinelStart() {
	alertsStore = make(map[uint64]map[uint64][][]interface{})
	alertsPrevState = make(map[uint64]map[uint64]uint)
	AlertsCycleTransferStatsStore = make(map[uint64]*model.CycleTransferStats)
	AlertsLock.Lock()
	if err := DB.Find(&Alerts).Error; err != nil {
		panic(err)
	}
	for _, alert := range Alerts {
		// 旧版本可能不存在通知组 为其添加默认值
		if alert.NotificationTag == "" {
			alert.NotificationTag = "default"
			DB.Save(alert)
		}
		alertsStore[alert.ID] = make(map[uint64][][]interface{})
		alertsPrevState[alert.ID] = make(map[uint64]uint)
		addCycleTransferStatsInfo(alert)
	}
	AlertsLock.Unlock()

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
	AlertsLock.Lock()
	defer AlertsLock.Unlock()
	delete(alertsStore, alert.ID)
	delete(alertsPrevState, alert.ID)
	var isEdit bool
	for i := 0; i < len(Alerts); i++ {
		if Alerts[i].ID == alert.ID {
			Alerts[i] = &alert
			isEdit = true
		}
	}
	if !isEdit {
		Alerts = append(Alerts, &alert)
	}
	alertsStore[alert.ID] = make(map[uint64][][]interface{})
	alertsPrevState[alert.ID] = make(map[uint64]uint)
	delete(AlertsCycleTransferStatsStore, alert.ID)
	addCycleTransferStatsInfo(&alert)
}

func OnDeleteAlert(id uint64) {
	AlertsLock.Lock()
	defer AlertsLock.Unlock()
	delete(alertsStore, id)
	delete(alertsPrevState, id)
	for i := 0; i < len(Alerts); i++ {
		if Alerts[i].ID == id {
			Alerts = append(Alerts[:i], Alerts[i+1:]...)
			i--
		}
	}
	delete(AlertsCycleTransferStatsStore, id)
}

// checkStatus 检查报警规则并发送报警
func checkStatus() {
	AlertsLock.RLock()
	defer AlertsLock.RUnlock()
	ServerLock.RLock()
	defer ServerLock.RUnlock()

	for _, alert := range Alerts {
		// 跳过未启用
		if !alert.Enabled() {
			continue
		}
		for _, server := range ServerList {
			// 监测点
			alertsStore[alert.ID][server.ID] = append(alertsStore[alert.
				ID][server.ID], alert.Snapshot(AlertsCycleTransferStatsStore[alert.ID], server, DB))
			// 发送通知，分为触发报警和恢复通知
			max, passed := alert.Check(alertsStore[alert.ID][server.ID])
			// 保存当前服务器状态信息
			curServer := model.Server{}
			copier.Copy(&curServer, server)
			if !passed {
				alertsPrevState[alert.ID][server.ID] = _RuleCheckFail
				message := fmt.Sprintf("[主机故障] %s(%s) 规则：%s", server.Name, IPDesensitize(server.Host.IP), alert.Name)
				go SendNotification(alert.NotificationTag, message, true, &curServer)
			} else {
				if alertsPrevState[alert.ID][server.ID] == _RuleCheckFail {
					message := fmt.Sprintf("[主机恢复] %s(%s) 规则：%s", server.Name, IPDesensitize(server.Host.IP), alert.Name)
					go SendNotification(alert.NotificationTag, message, true, &curServer)
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
