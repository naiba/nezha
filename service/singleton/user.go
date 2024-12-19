package singleton

import (
	"sync"

	"github.com/nezhahq/nezha/model"
	"gorm.io/gorm"
)

var (
	UserIdToAgentSecret map[uint64]string
	AgentSecretToUserId map[string]uint64

	UserLock sync.RWMutex
)

func initUser() {
	UserIdToAgentSecret = make(map[uint64]string)
	AgentSecretToUserId = make(map[string]uint64)

	var users []model.User
	DB.Find(&users)

	for _, u := range users {
		UserIdToAgentSecret[u.ID] = u.AgentSecret
		AgentSecretToUserId[u.AgentSecret] = u.ID
	}
}

func OnUserUpdate(u *model.User) {
	UserLock.Lock()
	defer UserLock.Unlock()

	if u == nil {
		return
	}

	UserIdToAgentSecret[u.ID] = u.AgentSecret
	AgentSecretToUserId[u.AgentSecret] = u.ID
}

func OnUserDelete(id []uint64) {
	UserLock.Lock()
	defer UserLock.Unlock()

	if len(id) < 1 {
		return
	}

	var (
		cron   bool
		server bool
	)

	for _, uid := range id {
		secret := UserIdToAgentSecret[uid]
		delete(AgentSecretToUserId, secret)
		delete(UserIdToAgentSecret, uid)

		CronLock.RLock()
		crons := model.FindUserID(CronList, uid)
		CronLock.RUnlock()

		cron = len(crons) > 0
		if cron {
			DB.Unscoped().Delete(&model.Cron{}, "id in (?)", crons)
			OnDeleteCron(crons)
		}

		SortedServerLock.RLock()
		servers := model.FindUserID(SortedServerList, uid)
		SortedServerLock.RUnlock()

		server = len(servers) > 0
		if server {
			DB.Transaction(func(tx *gorm.DB) error {
				if err := tx.Unscoped().Delete(&model.Server{}, "id in (?)", servers).Error; err != nil {
					return err
				}
				if err := tx.Unscoped().Delete(&model.ServerGroupServer{}, "server_id in (?)", servers).Error; err != nil {
					return err
				}
				return nil
			})

			AlertsLock.Lock()
			for _, sid := range servers {
				for _, alert := range Alerts {
					if AlertsCycleTransferStatsStore[alert.ID] != nil {
						delete(AlertsCycleTransferStatsStore[alert.ID].ServerName, sid)
						delete(AlertsCycleTransferStatsStore[alert.ID].Transfer, sid)
						delete(AlertsCycleTransferStatsStore[alert.ID].NextUpdate, sid)
					}
				}
			}
			DB.Unscoped().Delete(&model.Transfer{}, "server_id in (?)", servers)
			AlertsLock.Unlock()
			OnServerDelete(servers)
		}
	}

	if cron {
		UpdateCronList()
	}

	if server {
		ReSortServer()
	}
}
