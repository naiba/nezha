package dao

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/naiba/nezha/model"
	pb "github.com/naiba/nezha/proto"
)

var ServiceSentinelShared *ServiceSentinel

func NewServiceSentinel() {
	ServiceSentinelShared = &ServiceSentinel{
		serviceResponseChannel:              make(chan *pb.TaskResult, 200),
		serviceResponseDataStoreToday:       make(map[uint64][]model.MonitorHistory),
		lastStatus:                          make(map[uint64]string),
		serviceResponseDataStoreCurrentUp:   make(map[uint64]uint64),
		serviceResponseDataStoreCurrentDown: make(map[uint64]uint64),
		monitors:                            make(map[uint64]model.Monitor),
		latestDate:                          time.Now().Format("02-Jan-06"),
	}
	ServiceSentinelShared.OnMonitorUpdate()
	go ServiceSentinelShared.worker()
}

/*
   使用缓存 channel，处理上报的 Service 请求结果，然后判断是否需要报警
   需要记录上一次的状态信息
*/
type ServiceSentinel struct {
	latestDate                              string
	serviceResponseDataStoreTodaySavedIndex int
	serviceResponseDataStoreTodayLastSave   time.Time
	serviceResponseDataStoreLock            sync.RWMutex
	monitorsLock                            sync.RWMutex
	serviceResponseChannel                  chan *pb.TaskResult
	lastStatus                              map[uint64]string
	monitors                                map[uint64]model.Monitor
	serviceResponseDataStoreToday           map[uint64][]model.MonitorHistory
	serviceResponseDataStoreCurrentUp       map[uint64]uint64
	serviceResponseDataStoreCurrentDown     map[uint64]uint64
}

func (ss *ServiceSentinel) Dispatch(r *pb.TaskResult) {
	ss.serviceResponseChannel <- r
}

func (ss *ServiceSentinel) Monitors() []model.Monitor {
	ss.monitorsLock.RLock()
	defer ss.monitorsLock.RUnlock()
	var monitors []model.Monitor
	for _, v := range ss.monitors {
		monitors = append(monitors, v)
	}
	return monitors
}

func (ss *ServiceSentinel) OnMonitorUpdate() {
	var monitors []model.Monitor
	DB.Find(&monitors)
	ss.monitorsLock.Lock()
	defer ss.monitorsLock.Unlock()
	ss.monitors = make(map[uint64]model.Monitor)
	for i := 0; i < len(monitors); i++ {
		ss.monitors[monitors[i].ID] = monitors[i]
	}
}

func (ss *ServiceSentinel) OnMonitorDelete(id uint64) {
	ss.serviceResponseDataStoreLock.Lock()
	defer ss.serviceResponseDataStoreLock.Unlock()
	delete(ss.serviceResponseDataStoreCurrentDown, id)
	delete(ss.serviceResponseDataStoreCurrentUp, id)
	delete(ss.serviceResponseDataStoreToday, id)
	delete(ss.lastStatus, id)
	ss.monitorsLock.Lock()
	defer ss.monitorsLock.Unlock()
	delete(ss.monitors, id)
}

func (ss *ServiceSentinel) LoadStats() map[uint64]*model.ServiceItemResponse {
	var cached bool
	var msm map[uint64]*model.ServiceItemResponse
	data, has := Cache.Get(model.CacheKeyServicePage)
	if has {
		msm = data.(map[uint64]*model.ServiceItemResponse)
		cached = true
	}
	if !cached {
		msm = make(map[uint64]*model.ServiceItemResponse)
		var ms []model.Monitor
		DB.Find(&ms)
		year, month, day := time.Now().Date()
		today := time.Date(year, month, day, 0, 0, 0, 0, time.Local)
		var mhs []model.MonitorHistory
		DB.Where("created_at >= ? AND created_at < ?", today.AddDate(0, 0, -29), today).Find(&mhs)
		for i := 0; i < len(ms); i++ {
			msm[ms[i].ID] = &model.ServiceItemResponse{
				Monitor: ms[i],
				Delay:   &[30]float32{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				Up:      &[30]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				Down:    &[30]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			}
		}
		// 整合数据
		for i := 0; i < len(mhs); i++ {
			dayIndex := 28 - (int(today.Sub(mhs[i].CreatedAt).Hours()) / 24)
			if mhs[i].Successful {
				msm[mhs[i].MonitorID].TotalUp++
				msm[mhs[i].MonitorID].Delay[dayIndex] = (msm[mhs[i].MonitorID].Delay[dayIndex]*float32(msm[mhs[i].MonitorID].Up[dayIndex]) + mhs[i].Delay) / float32(msm[mhs[i].MonitorID].Up[dayIndex]+1)
				msm[mhs[i].MonitorID].Up[dayIndex]++
			} else {
				msm[mhs[i].MonitorID].TotalDown++
				msm[mhs[i].MonitorID].Down[dayIndex]++
			}
		}
		// 缓存一天
		Cache.Set(model.CacheKeyServicePage, msm, time.Until(time.Date(year, month, day, 23, 59, 59, 999, today.Location())))
	}
	// 最新一天的数据
	ss.serviceResponseDataStoreLock.RLock()
	defer ss.serviceResponseDataStoreLock.RUnlock()
	for k, v := range ss.serviceResponseDataStoreToday {
		if msm[k] == nil {
			msm[k] = &model.ServiceItemResponse{}
		}
		msm[k].Monitor = ss.monitors[k]
		for i := 0; i < len(v); i++ {
			if v[i].Successful {
				msm[k].Up[29]++
			} else {
				msm[k].Down[29]++
			}
			msm[k].Delay[29] = (msm[k].Delay[29]*float32(msm[k].Up[29]) + v[i].Delay) / float32(msm[k].Up[29]+1)
		}
	}
	// 最后20分钟的状态 与 monitor 对象填充
	for k, v := range ss.serviceResponseDataStoreCurrentDown {
		msm[k].CurrentDown = v
	}
	for k, v := range ss.serviceResponseDataStoreCurrentUp {
		msm[k].CurrentUp = v
	}
	return msm
}

func getStateStr(percent uint64) string {
	if percent == 0 {
		return "无数据"
	}
	if percent > 95 {
		return "良好"
	}
	if percent > 80 {
		return "低可用"
	}
	return "故障"
}

func (ss *ServiceSentinel) worker() {
	for r := range ss.serviceResponseChannel {
		mh := model.PB2MonitorHistory(r)
		ss.serviceResponseDataStoreLock.Lock()
		// 先查看是否到下一天
		nowDate := time.Now().Format("02-Jan-06")
		if nowDate != ss.latestDate {
			ss.latestDate = nowDate
			dataToSave := ss.serviceResponseDataStoreToday[mh.MonitorID][ss.serviceResponseDataStoreTodaySavedIndex:]
			if err := DB.Create(&dataToSave).Error; err != nil {
				log.Println(err)
			}
			ss.serviceResponseDataStoreTodaySavedIndex = 0
			ss.serviceResponseDataStoreToday[mh.MonitorID] = []model.MonitorHistory{}
			ss.serviceResponseDataStoreCurrentDown[mh.MonitorID] = 0
			ss.serviceResponseDataStoreCurrentUp[mh.MonitorID] = 0
			ss.serviceResponseDataStoreTodayLastSave = time.Now()
		}
		// 储存至当日数据
		ss.serviceResponseDataStoreToday[mh.MonitorID] = append(ss.serviceResponseDataStoreToday[mh.MonitorID], mh)
		// 每20分钟入库一次
		if time.Now().After(ss.serviceResponseDataStoreTodayLastSave.Add(time.Minute * 20)) {
			ss.serviceResponseDataStoreTodayLastSave = time.Now()
			dataToSave := ss.serviceResponseDataStoreToday[mh.MonitorID][ss.serviceResponseDataStoreTodaySavedIndex:]
			if err := DB.Create(&dataToSave).Error; err != nil {
				log.Println(err)
			}
			ss.serviceResponseDataStoreTodaySavedIndex = len(ss.serviceResponseDataStoreToday[mh.MonitorID])
		}
		// 更新当前状态
		ss.serviceResponseDataStoreCurrentUp[mh.MonitorID] = 0
		ss.serviceResponseDataStoreCurrentDown[mh.MonitorID] = 0
		for i := len(ss.serviceResponseDataStoreToday[mh.MonitorID]) - 1; i >= 0 && i >= len(ss.serviceResponseDataStoreToday[mh.MonitorID])-20; i-- {
			if ss.serviceResponseDataStoreToday[mh.MonitorID][i].Successful {
				ss.serviceResponseDataStoreCurrentUp[mh.MonitorID]++
			} else {
				ss.serviceResponseDataStoreCurrentDown[mh.MonitorID]++
			}
		}
		stateStr := getStateStr(ss.serviceResponseDataStoreCurrentUp[mh.MonitorID] * 100 / (ss.serviceResponseDataStoreCurrentDown[mh.MonitorID] + ss.serviceResponseDataStoreCurrentUp[mh.MonitorID]))
		if stateStr == "故障" || stateStr != ss.lastStatus[mh.MonitorID] {
			ss.monitorsLock.RLock()
			isSendNotification := (ss.lastStatus[mh.MonitorID] != "" || stateStr == "故障") && ss.monitors[mh.MonitorID].Notify
			ss.lastStatus[mh.MonitorID] = stateStr
			if isSendNotification {
				SendNotification(fmt.Sprintf("服务监控：%s 服务状态：%s", ss.monitors[mh.MonitorID].Name, stateStr), true)
			}
			ss.monitorsLock.RUnlock()
		}
		ss.serviceResponseDataStoreLock.Unlock()
		// SSL 证书报警
		var errMsg string
		if strings.HasPrefix(r.GetData(), "SSL证书错误：") {
			// 排除 i/o timeont、connection timeout、EOF 错误
			if !strings.HasSuffix(r.GetData(), "timeout") &&
				!strings.HasSuffix(r.GetData(), "EOF") &&
				!strings.HasSuffix(r.GetData(), "timed out") {
				errMsg = r.GetData()
			}
		} else {
			var last model.MonitorHistory
			var newCert = strings.Split(r.GetData(), "|")
			if len(newCert) > 1 {
				expiresNew, _ := time.Parse("2006-01-02 15:04:05 -0700 MST", newCert[1])
				// 证书过期提醒
				if expiresNew.Before(time.Now().AddDate(0, 0, 7)) {
					errMsg = fmt.Sprintf(
						"SSL证书将在七天内过期，过期时间：%s。",
						expiresNew.Format("2006-01-02 15:04:05"))
				}
				// 证书变更提醒
				if err := DB.Where("monitor_id = ? AND data LIKE ?", r.GetId(), "%|%").Order("id DESC").First(&last).Error; err == nil {
					var oldCert = strings.Split(last.Data, "|")
					var expiresOld time.Time
					if len(oldCert) > 1 {
						expiresOld, _ = time.Parse("2006-01-02 15:04:05 -0700 MST", oldCert[1])
					}
					if last.Data != "" && oldCert[0] != newCert[0] && !expiresNew.Equal(expiresOld) {
						errMsg = fmt.Sprintf(
							"SSL证书变更，旧：%s, %s 过期；新：%s, %s 过期。",
							oldCert[0], expiresOld.Format("2006-01-02 15:04:05"), newCert[0], expiresNew.Format("2006-01-02 15:04:05"))
					}
				}
			}
		}
		if errMsg != "" {
			ss.monitorsLock.RLock()
			SendNotification(fmt.Sprintf("服务监控：%s %s", ss.monitors[mh.MonitorID].Name, errMsg), true)
			ss.monitorsLock.RUnlock()
		}
	}
}
