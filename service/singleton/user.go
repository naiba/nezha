package singleton

import (
	"sync"

	"github.com/nezhahq/nezha/model"
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

	for _, uid := range id {
		secret := UserIdToAgentSecret[uid]
		delete(AgentSecretToUserId, secret)
		delete(UserIdToAgentSecret, uid)
	}
}
