package singleton

import (
	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/pkg/utils"
	"log"
	"time"
)

/*
	该文件保存了一些工具函数
	RecordTransferHourlyUsage	对流量记录进行打点
	CleanMonitorHistory			清理无效或过时的 监控记录 和 流量记录
	IPDesensitize				根据用户设置选择是否对IP进行打码处理 返回处理后的IP(关闭打码则返回原IP)
*/

// RecordTransferHourlyUsage 对流量记录进行打点
func RecordTransferHourlyUsage() {
	ServerLock.Lock()
	defer ServerLock.Unlock()
	now := time.Now()
	nowTrimSeconds := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.Local)
	var txs []model.Transfer
	for id, server := range ServerList {
		tx := model.Transfer{
			ServerID: id,
			In:       server.State.NetInTransfer - uint64(server.PrevHourlyTransferIn),
			Out:      server.State.NetOutTransfer - uint64(server.PrevHourlyTransferOut),
		}
		if tx.In == 0 && tx.Out == 0 {
			continue
		}
		server.PrevHourlyTransferIn = int64(server.State.NetInTransfer)
		server.PrevHourlyTransferOut = int64(server.State.NetOutTransfer)
		tx.CreatedAt = nowTrimSeconds
		txs = append(txs, tx)
	}
	if len(txs) == 0 {
		return
	}
	log.Println("NEZHA>> Cron 流量统计入库", len(txs), DB.Create(txs).Error)
}

// CleanMonitorHistory 清理无效或过时的 监控记录 和 流量记录
func CleanMonitorHistory() {
	// 清理已被删除的服务器的监控记录与流量记录
	DB.Unscoped().Delete(&model.MonitorHistory{}, "created_at < ? OR monitor_id NOT IN (SELECT `id` FROM monitors)", time.Now().AddDate(0, 0, -30))
	DB.Unscoped().Delete(&model.Transfer{}, "server_id NOT IN (SELECT `id` FROM servers)")
	// 计算可清理流量记录的时长
	var allServerKeep time.Time
	specialServerKeep := make(map[uint64]time.Time)
	var specialServerIDs []uint64
	var alerts []model.AlertRule
	DB.Find(&alerts)
	for i := 0; i < len(alerts); i++ {
		for j := 0; j < len(alerts[i].Rules); j++ {
			// 是不是流量记录规则
			if !alerts[i].Rules[j].IsTransferDurationRule() {
				continue
			}
			dataCouldRemoveBefore := alerts[i].Rules[j].GetTransferDurationStart()
			// 判断规则影响的机器范围
			if alerts[i].Rules[j].Cover == model.RuleCoverAll {
				// 更新全局可以清理的数据点
				if allServerKeep.IsZero() || allServerKeep.After(dataCouldRemoveBefore) {
					allServerKeep = dataCouldRemoveBefore
				}
			} else {
				// 更新特定机器可以清理数据点
				for id := range alerts[i].Rules[j].Ignore {
					if specialServerKeep[id].IsZero() || specialServerKeep[id].After(dataCouldRemoveBefore) {
						specialServerKeep[id] = dataCouldRemoveBefore
						specialServerIDs = append(specialServerIDs, id)
					}
				}
			}
		}
	}
	for id, couldRemove := range specialServerKeep {
		DB.Unscoped().Delete(&model.Transfer{}, "server_id = ? AND created_at < ?", id, couldRemove)
	}
	if allServerKeep.IsZero() {
		DB.Unscoped().Delete(&model.Transfer{}, "server_id NOT IN (?)", specialServerIDs)
	} else {
		DB.Unscoped().Delete(&model.Transfer{}, "server_id NOT IN (?) AND created_at < ?", specialServerIDs, allServerKeep)
	}
}

// IPDesensitize 根据设置选择是否对IP进行打码处理 返回处理后的IP(关闭打码则返回原IP)
func IPDesensitize(ip string) string {
	if Conf.EnablePlainIPInNotification {
		return ip
	}
	return utils.IPDesensitize(ip)
}
