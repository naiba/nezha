package singleton

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/naiba/nezha/model"
)

const (
	firstNotificationDelay = time.Minute * 15
	defaultGroupID         = 0
)

// 通知方式
var (
	NotificationList      map[uint64]map[uint64]*model.Notification // [NotificationGroupID][NotificationID] -> model.Notification
	NotificationIDToGroup map[uint64]uint64                         // [NotificationID] -> NotificationGroupID

	NotificationMap   map[uint64]*model.Notification
	NotificationGroup map[uint64]string // [NotificationGroupID] -> [NotificationGroupName]

	notificationsLock     sync.RWMutex
	notificationGroupLock sync.RWMutex
)

// InitNotification 初始化 GroupID <-> ID <-> Notification 的映射
func InitNotification() {
	NotificationList = make(map[uint64]map[uint64]*model.Notification)
	NotificationIDToGroup = make(map[uint64]uint64)
	// 默认分组
	NotificationList[defaultGroupID] = make(map[uint64]*model.Notification)
	NotificationGroup[defaultGroupID] = "default"
}

// loadNotifications 从 DB 初始化通知方式相关参数
func loadNotifications() {
	InitNotification()
	notificationsLock.Lock()
	defer notificationsLock.Unlock()

	groupNotifications := make(map[uint64][]uint64)
	var ngn []model.NotificationGroupNotification
	if err := DB.Find(&ngn).Error; err != nil {
		panic(err)
	}

	for _, n := range ngn {
		groupNotifications[n.NotificationGroupID] = append(groupNotifications[n.NotificationGroupID], n.NotificationID)
	}

	var notifications []model.Notification
	if err := DB.Find(&notifications).Error; err != nil {
		panic(err)
	}

	NotificationMap = make(map[uint64]*model.Notification, len(notifications))
	for i := range notifications {
		NotificationMap[notifications[i].ID] = &notifications[i]
	}

	for gid, nids := range groupNotifications {
		NotificationList[gid] = make(map[uint64]*model.Notification)
		for _, nid := range nids {
			if n, ok := NotificationMap[nid]; ok {
				NotificationList[gid][n.ID] = n
				NotificationIDToGroup[n.ID] = gid
			}
		}
	}

	for _, n := range notifications {
		if _, ok := NotificationIDToGroup[n.ID]; !ok {
			ncopy := n
			NotificationList[defaultGroupID][n.ID] = &ncopy
			NotificationIDToGroup[n.ID] = defaultGroupID
		}
	}
}

// OnRefreshOrAddNotificationGroup 刷新通知方式组相关参数
func OnRefreshOrAddNotificationGroup(ng *model.NotificationGroup, ngn []uint64) {
	notificationsLock.Lock()
	defer notificationsLock.Unlock()

	notificationGroupLock.Lock()
	defer notificationGroupLock.Unlock()
	var isEdit bool
	if _, ok := NotificationGroup[ng.ID]; ok {
		isEdit = true
	}

	if !isEdit {
		AddNotificationGroupToList(ng, ngn)
	} else {
		UpdateNotificationGroupInList(ng, ngn)
	}
}

// AddNotificationGroupToList 添加通知方式组到map中
func AddNotificationGroupToList(ng *model.NotificationGroup, ngn []uint64) {
	NotificationGroup[ng.ID] = ng.Name

	NotificationList[ng.ID] = make(map[uint64]*model.Notification, len(ngn))

	for _, n := range ngn {
		// 确认新的绑定关系，从默认分组中移出
		delete(NotificationList[defaultGroupID], n)

		NotificationIDToGroup[n] = ng.ID
		NotificationList[ng.ID][n] = NotificationMap[n]
	}
}

// UpdateNotificationGroupInList 在 map 中更新通知方式组
func UpdateNotificationGroupInList(ng *model.NotificationGroup, ngn []uint64) {
	NotificationGroup[ng.ID] = ng.Name

	oldList := make(map[uint64]struct{})
	for nid := range NotificationList[ng.ID] {
		oldList[nid] = struct{}{}
	}

	NotificationList[ng.ID] = make(map[uint64]*model.Notification)
	for _, nid := range ngn {
		NotificationList[ng.ID][nid] = NotificationMap[nid]
		NotificationIDToGroup[nid] = ng.ID
	}

	for oldID := range oldList {
		if _, exists := NotificationList[ng.ID][oldID]; !exists {
			delete(NotificationIDToGroup, oldID)
			// 删除的 ID 回落到默认分组
			NotificationIDToGroup[oldID] = defaultGroupID
			NotificationList[defaultGroupID][oldID] = NotificationMap[oldID]
		}
	}
}

// UpdateNotificationGroupInList 删除通知方式组
func OnDeleteNotificationGroup(gids []uint64) {
	notificationsLock.Lock()
	defer notificationsLock.Unlock()

	for _, gid := range gids {
		delete(NotificationGroup, gid)

		// 删除的通知组下所有通知转移到默认分组
		for nid, notification := range NotificationList[gid] {
			NotificationList[defaultGroupID][nid] = notification
			NotificationIDToGroup[nid] = defaultGroupID
		}

		delete(NotificationList, gid)
	}
}

// OnRefreshOrAddNotification 刷新通知方式相关参数
func OnRefreshOrAddNotification(n *model.Notification) {
	notificationsLock.Lock()
	defer notificationsLock.Unlock()

	var isEdit bool
	gid, ok := NotificationIDToGroup[n.ID]
	if ok {
		isEdit = true
	}
	if !isEdit {
		AddNotificationToList(n)
	} else {
		UpdateNotificationInList(n, gid)
	}
}

// AddNotificationToList 添加通知方式到map中
func AddNotificationToList(n *model.Notification) {
	// 当前 Tag 不存在，创建对应该 Tag 的 子 map 后再添加
	//if _, ok := NotificationList[n.Tag]; !ok {
	//	NotificationList[n.Tag] = make(map[uint64]*model.Notification)
	//}

	// 新创建的通知方式添加到默认分组
	NotificationMap[n.ID] = n
	NotificationList[defaultGroupID][n.ID] = n
	NotificationIDToGroup[n.ID] = defaultGroupID
}

// UpdateNotificationInList 在 map 中更新通知方式
func UpdateNotificationInList(n *model.Notification, gid uint64) {
	NotificationMap[n.ID] = n
	NotificationList[gid][n.ID] = n
}

// OnDeleteNotification 在map和表中删除通知方式
func OnDeleteNotification(id []uint64) {
	notificationsLock.Lock()
	defer notificationsLock.Unlock()

	for _, i := range id {
		gid := NotificationIDToGroup[i]
		delete(NotificationList[gid], i)
		delete(NotificationIDToGroup, i)
	}

	if err := DB.Delete(&model.NotificationGroupNotification{}, "notification_id in (?)", id).Error; err != nil {
		log.Println(err)
	}
}

func UnMuteNotification(notificationGroupID uint64, muteLabel *string) {
	fullMuteLabel := *NotificationMuteLabel.AppendNotificationGroupName(muteLabel, notificationGroupID)
	Cache.Delete(fullMuteLabel)
}

// SendNotification 向指定的通知方式组的所有通知方式发送通知
func SendNotification(notificationGroupID uint64, desc string, muteLabel *string, ext ...*model.Server) {
	if muteLabel != nil {
		// 将通知方式组名称加入静音标志
		muteLabel := *NotificationMuteLabel.AppendNotificationGroupName(muteLabel, notificationGroupID)
		// 通知防骚扰策略
		var flag bool
		if cacheN, has := Cache.Get(muteLabel); has {
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
				Cache.Set(muteLabel, nHistory, nHistory.Duration+time.Minute*10)
			}
		} else {
			// 新提醒直接通知
			flag = true
			Cache.Set(muteLabel, NotificationHistory{
				Duration: firstNotificationDelay,
				Until:    time.Now().Add(firstNotificationDelay),
			}, firstNotificationDelay+time.Minute*10)
		}

		if !flag {
			if Conf.Debug {
				log.Println("NEZHA>> 静音的重复通知：", desc, muteLabel)
			}
			return
		}
	}
	// 向该通知方式组的所有通知方式发出通知
	notificationsLock.RLock()
	defer notificationsLock.RUnlock()
	for _, n := range NotificationList[notificationGroupID] {
		log.Println("NEZHA>> 尝试通知", n.Name)
	}
	for _, n := range NotificationList[notificationGroupID] {
		ns := model.NotificationServerBundle{
			Notification: n,
			Server:       nil,
			Loc:          Loc,
		}
		if len(ext) > 0 {
			ns.Server = ext[0]
		}
		if err := ns.Send(desc); err != nil {
			log.Println("NEZHA>> 向 ", n.Name, " 发送通知失败：", err)
		} else {
			log.Println("NEZHA>> 向 ", n.Name, " 发送通知成功：")
		}
	}
}

type _NotificationMuteLabel struct{}

var NotificationMuteLabel _NotificationMuteLabel

func (_NotificationMuteLabel) IPChanged(serverId uint64) *string {
	label := fmt.Sprintf("bf::ic-%d", serverId)
	return &label
}

func (_NotificationMuteLabel) ServerIncident(alertId uint64, serverId uint64) *string {
	label := fmt.Sprintf("bf::sei-%d-%d", alertId, serverId)
	return &label
}

func (_NotificationMuteLabel) ServerIncidentResolved(alertId uint64, serverId uint64) *string {
	label := fmt.Sprintf("bf::seir-%d-%d", alertId, serverId)
	return &label
}

func (_NotificationMuteLabel) AppendNotificationGroupName(label *string, notificationGroupID uint64) *string {
	notificationGroupLock.RLock()
	defer notificationGroupLock.RUnlock()
	newLabel := fmt.Sprintf("%s:%s", *label, NotificationGroup[notificationGroupID])
	return &newLabel
}

func (_NotificationMuteLabel) ServiceLatencyMin(serviceId uint64) *string {
	label := fmt.Sprintf("bf::sln-%d", serviceId)
	return &label
}

func (_NotificationMuteLabel) ServiceLatencyMax(serviceId uint64) *string {
	label := fmt.Sprintf("bf::slm-%d", serviceId)
	return &label
}

func (_NotificationMuteLabel) ServiceStateChanged(serviceId uint64) *string {
	label := fmt.Sprintf("bf::ssc-%d", serviceId)
	return &label
}

func (_NotificationMuteLabel) ServiceSSL(serviceId uint64, extraInfo string) *string {
	label := fmt.Sprintf("bf::sssl-%d-%s", serviceId, extraInfo)
	return &label
}
