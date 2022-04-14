package singleton

import (
	"crypto/md5" // #nosec
	"encoding/hex"
	"log"
	"sync"
	"time"

	"github.com/naiba/nezha/model"
)

const firstNotificationDelay = time.Minute * 15

// 通知方式
var (
	NotificationList    map[string]map[uint64]*model.Notification // [NotificationMethodTag][NotificationID] -> model.Notification
	NotificationIDToTag map[uint64]string                         // [NotificationID] -> NotificationTag
	notificationsLock   sync.RWMutex
)

// InitNotification 初始化 Tag <-> ID <-> Notification 的映射
func InitNotification() {
	NotificationList = make(map[string]map[uint64]*model.Notification)
	NotificationIDToTag = make(map[uint64]string)
}

// LoadNotifications 从 DB 初始化通知方式相关参数
func LoadNotifications() {
	InitNotification()
	notificationsLock.Lock()
	defer notificationsLock.Unlock()

	var notifications []model.Notification
	if err := DB.Find(&notifications).Error; err != nil {
		panic(err)
	}
	for _, n := range notifications {
		// 旧版本的Tag可能不存在 自动设置为默认值
		if n.Tag == "" {
			SetDefaultNotificationTagInDB(&n)
		}
		AddNotificationToList(&n)
	}
}

// SetDefaultNotificationTagInDB 设置默认通知方式的 Tag
func SetDefaultNotificationTagInDB(n *model.Notification) {
	n.Tag = "default"
	if err := DB.Save(n).Error; err != nil {
		log.Println("[ERROR]", err)
	}
}

// OnRefreshOrAddNotification 刷新通知方式相关参数
func OnRefreshOrAddNotification(n *model.Notification) {
	notificationsLock.Lock()
	defer notificationsLock.Unlock()

	var isEdit bool
	if _, ok := NotificationList[n.Tag][n.ID]; ok {
		isEdit = true
	}
	if !isEdit {
		AddNotificationToList(n)
	} else {
		UpdateNotificationInList(n)
	}
}

// AddNotificationToList 添加通知方式到map中
func AddNotificationToList(n *model.Notification) {
	notificationsLock.Lock()
	defer notificationsLock.Unlock()

	// 当前 Tag 不存在，创建对应该 Tag 的 子 map 后再添加
	if _, ok := NotificationList[n.Tag]; !ok {
		NotificationList[n.Tag] = make(map[uint64]*model.Notification)
	}
	NotificationList[n.Tag][n.ID] = n
	NotificationIDToTag[n.ID] = n.Tag

}

// UpdateNotificationInList 在 map 中更新通知方式
func UpdateNotificationInList(n *model.Notification) {
	notificationsLock.Lock()
	defer notificationsLock.Unlock()

	NotificationList[n.Tag][n.ID] = n
}

// OnDeleteNotification 在map中删除通知方式
func OnDeleteNotification(id uint64) {
	notificationsLock.Lock()
	defer notificationsLock.Unlock()

	delete(NotificationList[NotificationIDToTag[id]], id)
	delete(NotificationIDToTag, id)
}

// SendNotification 向指定的通知方式组的所有通知方式发送通知
func SendNotification(notificationTag string, desc string, mutable bool) {
	if mutable {
		// 通知防骚扰策略
		nID := hex.EncodeToString(md5.New().Sum([]byte(desc))) // #nosec
		var flag bool
		if cacheN, has := Cache.Get(nID); has {
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
				Cache.Set(nID, nHistory, nHistory.Duration+time.Minute*10)
			}
		} else {
			// 新提醒直接通知
			flag = true
			Cache.Set(nID, NotificationHistory{
				Duration: firstNotificationDelay,
				Until:    time.Now().Add(firstNotificationDelay),
			}, firstNotificationDelay+time.Minute*10)
		}

		if !flag {
			if Conf.Debug {
				log.Println("NEZHA>> 静音的重复通知：", desc, mutable)
			}
			return
		}
	}
	// 向该通知方式组的所有通知方式发出通知
	notificationsLock.RLock()
	defer notificationsLock.RUnlock()

	for _, n := range NotificationList[notificationTag] {
		if err := n.Send(desc); err != nil {
			log.Println("NEZHA>> 发送通知失败：", err)
		}
	}
}
