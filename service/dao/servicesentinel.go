package dao

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/naiba/nezha/model"
	pb "github.com/naiba/nezha/proto"
)

const _CurrentStatusSize = 30 // 统计 15 分钟内的数据为当前状态

var ServiceSentinelShared *ServiceSentinel

type ReportData struct {
	Data     *pb.TaskResult
	Reporter uint64
}

type _TodayStatsOfMonitor struct {
	Up    int
	Down  int
	Delay float32
}

func NewServiceSentinel(serviceSentinelDispatchBus chan<- model.Monitor) {
	ServiceSentinelShared = &ServiceSentinel{
		serviceReportChannel:                make(chan ReportData, 200),
		serviceStatusToday:                  make(map[uint64]*_TodayStatsOfMonitor),
		serviceCurrentStatusIndex:           make(map[uint64]int),
		serviceCurrentStatusData:            make(map[uint64][]model.MonitorHistory),
		latestDate:                          make(map[uint64]string),
		lastStatus:                          make(map[uint64]string),
		serviceResponseDataStoreCurrentUp:   make(map[uint64]uint64),
		serviceResponseDataStoreCurrentDown: make(map[uint64]uint64),
		monitors:                            make(map[uint64]*model.Monitor),
		sslCertCache:                        make(map[uint64]string),
		// 30天数据缓存
		monthlyStatus: make(map[uint64]*model.ServiceItemResponse),
		dispatchBus:   serviceSentinelDispatchBus,
	}
	ServiceSentinelShared.loadMonitorHistory()

	year, month, day := time.Now().Date()
	today := time.Date(year, month, day, 0, 0, 0, 0, time.Local)

	var mhs []model.MonitorHistory
	// 加载当日记录
	DB.Where("created_at >= ?", today).Find(&mhs)
	totalDelay := make(map[uint64]float32)
	for i := 0; i < len(mhs); i++ {
		if mhs[i].Successful {
			ServiceSentinelShared.serviceStatusToday[mhs[i].MonitorID].Up++
			totalDelay[mhs[i].MonitorID] += mhs[i].Delay
		} else {
			ServiceSentinelShared.serviceStatusToday[mhs[i].MonitorID].Down++
		}
	}
	for id, delay := range totalDelay {
		ServiceSentinelShared.serviceStatusToday[id].Delay = delay / float32(ServiceSentinelShared.serviceStatusToday[id].Up)
	}

	// 更新入库时间及当日数据入库游标
	for k := range ServiceSentinelShared.monitors {
		ServiceSentinelShared.latestDate[k] = time.Now().Format("02-Jan-06")
	}

	go ServiceSentinelShared.worker()

	// 每日将游标往后推一天
	_, err := Cron.AddFunc("0 0 0 * * *", ServiceSentinelShared.refreshMonthlyServiceStatus)
	if err != nil {
		panic(err)
	}
}

/*
   使用缓存 channel，处理上报的 Service 请求结果，然后判断是否需要报警
   需要记录上一次的状态信息
*/
type ServiceSentinel struct {
	serviceResponseDataStoreLock        sync.RWMutex
	monitorsLock                        sync.RWMutex
	serviceReportChannel                chan ReportData
	serviceStatusToday                  map[uint64]*_TodayStatsOfMonitor
	serviceCurrentStatusIndex           map[uint64]int
	serviceCurrentStatusData            map[uint64][]model.MonitorHistory
	latestDate                          map[uint64]string
	lastStatus                          map[uint64]string
	serviceResponseDataStoreCurrentUp   map[uint64]uint64
	serviceResponseDataStoreCurrentDown map[uint64]uint64
	monitors                            map[uint64]*model.Monitor
	sslCertCache                        map[uint64]string
	// 30天数据缓存
	monthlyStatusLock sync.Mutex
	monthlyStatus     map[uint64]*model.ServiceItemResponse
	// 服务监控调度计划任务
	dispatchBus chan<- model.Monitor
}

func (ss *ServiceSentinel) refreshMonthlyServiceStatus() {
	// 刷新数据防止无人访问
	ss.LoadStats()
	// 将数据往前刷一天
	ss.monthlyStatusLock.Lock()
	defer ss.monthlyStatusLock.Unlock()
	for _, v := range ss.monthlyStatus {
		for i := 0; i < len(v.Up)-1; i++ {
			v.Up[i] = v.Up[i+1]
			v.Down[i] = v.Down[i+1]
			v.Delay[i] = v.Delay[i+1]
		}
	}
}

func (ss *ServiceSentinel) Dispatch(r ReportData) {
	ss.serviceReportChannel <- r
}

func (ss *ServiceSentinel) Monitors() []*model.Monitor {
	ss.monitorsLock.RLock()
	defer ss.monitorsLock.RUnlock()
	var monitors []*model.Monitor
	for _, v := range ss.monitors {
		monitors = append(monitors, v)
	}
	sort.SliceStable(monitors, func(i, j int) bool {
		return monitors[i].ID < monitors[j].ID
	})
	return monitors
}

func (ss *ServiceSentinel) loadMonitorHistory() {
	var monitors []*model.Monitor
	DB.Find(&monitors)
	var err error
	ss.monitorsLock.Lock()
	defer ss.monitorsLock.Unlock()
	ss.monitors = make(map[uint64]*model.Monitor)
	for i := 0; i < len(monitors); i++ {
		task := *monitors[i]
		monitors[i].CronJobID, err = Cron.AddFunc(task.CronSpec(), func() {
			ss.dispatchBus <- task
		})
		if err != nil {
			panic(err)
		}
		ss.monitors[monitors[i].ID] = monitors[i]
		ss.serviceCurrentStatusData[monitors[i].ID] = make([]model.MonitorHistory, _CurrentStatusSize)
		ss.serviceStatusToday[monitors[i].ID] = &_TodayStatsOfMonitor{}
	}

	year, month, day := time.Now().Date()
	today := time.Date(year, month, day, 0, 0, 0, 0, time.Local)

	ss.monthlyStatusLock.Lock()
	defer ss.monthlyStatusLock.Unlock()
	for i := 0; i < len(monitors); i++ {
		ServiceSentinelShared.monthlyStatus[monitors[i].ID] = &model.ServiceItemResponse{
			Monitor: monitors[i],
			Delay:   &[30]float32{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			Up:      &[30]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			Down:    &[30]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		}
	}

	// 加载历史记录
	var mhs []model.MonitorHistory
	DB.Where("created_at >= ? AND created_at < ?", today.AddDate(0, 0, -29), today).Find(&mhs)
	for i := 0; i < len(mhs); i++ {
		dayIndex := 28 - (int(today.Sub(mhs[i].CreatedAt).Hours()) / 24)
		if mhs[i].Successful {
			ServiceSentinelShared.monthlyStatus[mhs[i].MonitorID].TotalUp++
			ServiceSentinelShared.monthlyStatus[mhs[i].MonitorID].Delay[dayIndex] = (ServiceSentinelShared.monthlyStatus[mhs[i].MonitorID].Delay[dayIndex]*float32(ss.monthlyStatus[mhs[i].MonitorID].Up[dayIndex]) + mhs[i].Delay) / float32(ss.monthlyStatus[mhs[i].MonitorID].Up[dayIndex]+1)
			ServiceSentinelShared.monthlyStatus[mhs[i].MonitorID].Up[dayIndex]++
		} else {
			ServiceSentinelShared.monthlyStatus[mhs[i].MonitorID].TotalDown++
			ServiceSentinelShared.monthlyStatus[mhs[i].MonitorID].Down[dayIndex]++
		}
	}
}

func (ss *ServiceSentinel) OnMonitorUpdate(m model.Monitor) error {
	ss.monitorsLock.Lock()
	defer ss.monitorsLock.Unlock()
	var err error
	// 写入新任务
	m.CronJobID, err = Cron.AddFunc(m.CronSpec(), func() {
		ss.dispatchBus <- m
	})
	if err != nil {
		return err
	}
	if ss.monitors[m.ID] != nil {
		// 停掉旧任务
		Cron.Remove(ss.monitors[m.ID].CronJobID)
	} else {
		// 新任务初始化数据
		ss.monthlyStatusLock.Lock()
		defer ss.monthlyStatusLock.Unlock()
		ss.monthlyStatus[m.ID] = &model.ServiceItemResponse{
			Monitor: &m,
			Delay:   &[30]float32{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			Up:      &[30]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			Down:    &[30]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		}
		ss.serviceResponseDataStoreLock.Lock()
		defer ss.serviceResponseDataStoreLock.Unlock()
		ss.serviceCurrentStatusData[m.ID] = make([]model.MonitorHistory, _CurrentStatusSize)
		ss.serviceStatusToday[m.ID] = &_TodayStatsOfMonitor{}
	}
	// 更新这个任务
	ss.monitors[m.ID] = &m
	return nil
}

func (ss *ServiceSentinel) OnMonitorDelete(id uint64) {
	ss.serviceResponseDataStoreLock.Lock()
	defer ss.serviceResponseDataStoreLock.Unlock()
	delete(ss.serviceCurrentStatusIndex, id)
	delete(ss.serviceCurrentStatusData, id)
	delete(ss.latestDate, id)
	delete(ss.lastStatus, id)
	delete(ss.serviceResponseDataStoreCurrentUp, id)
	delete(ss.serviceResponseDataStoreCurrentDown, id)
	delete(ss.sslCertCache, id)
	ss.monitorsLock.Lock()
	defer ss.monitorsLock.Unlock()
	// 停掉定时任务
	Cron.Remove(ss.monitors[id].CronJobID)
	delete(ss.monitors, id)
	ss.monthlyStatusLock.Lock()
	defer ss.monthlyStatusLock.Unlock()
	delete(ss.monthlyStatus, id)
}

func (ss *ServiceSentinel) LoadStats() map[uint64]*model.ServiceItemResponse {
	// 刷新最新一天的数据
	ss.serviceResponseDataStoreLock.RLock()
	defer ss.serviceResponseDataStoreLock.RUnlock()
	ss.monthlyStatusLock.Lock()
	defer ss.monthlyStatusLock.Unlock()
	for k := range ss.monitors {
		ss.monthlyStatus[k].Monitor = ss.monitors[k]
		v := ss.serviceStatusToday[k]
		ss.monthlyStatus[k].Up[29] = v.Up
		ss.monthlyStatus[k].Down[29] = v.Down
		ss.monthlyStatus[k].TotalUp += uint64(v.Up)
		ss.monthlyStatus[k].TotalDown += uint64(v.Down)
		ss.monthlyStatus[k].Delay[29] = v.Delay
	}
	// 最后 5 分钟的状态 与 monitor 对象填充
	for k, v := range ss.serviceResponseDataStoreCurrentDown {
		ss.monthlyStatus[k].CurrentDown = v
	}
	for k, v := range ss.serviceResponseDataStoreCurrentUp {
		ss.monthlyStatus[k].CurrentUp = v
	}
	return ss.monthlyStatus
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
	for r := range ss.serviceReportChannel {
		if ss.monitors[r.Data.GetId()] == nil || ss.monitors[r.Data.GetId()].ID == 0 {
			log.Printf("NEZAH>> 错误的服务监控上报 %+v", r)
			continue
		}
		mh := model.PB2MonitorHistory(r.Data)
		ss.serviceResponseDataStoreLock.Lock()
		// 先查看是否到下一天
		nowDate := time.Now().Format("02-Jan-06")
		if nowDate != ss.latestDate[mh.MonitorID] {
			// 清理前一天数据
			ss.latestDate[mh.MonitorID] = nowDate
			ss.serviceResponseDataStoreCurrentUp[mh.MonitorID] = 0
			ss.serviceResponseDataStoreCurrentDown[mh.MonitorID] = 0
			ss.serviceStatusToday[mh.MonitorID].Delay = 0
			ss.serviceStatusToday[mh.MonitorID].Up = 0
			ss.serviceStatusToday[mh.MonitorID].Down = 0
		}
		// 写入当天状态
		if mh.Successful {
			ss.serviceStatusToday[mh.MonitorID].Delay = (ss.serviceStatusToday[mh.
				MonitorID].Delay*float32(ss.serviceStatusToday[mh.MonitorID].Up) +
				mh.Delay) / float32(ss.serviceStatusToday[mh.MonitorID].Up+1)
			ss.serviceStatusToday[mh.MonitorID].Up++
		} else {
			ss.serviceStatusToday[mh.MonitorID].Down++
		}
		// 写入当前数据
		ss.serviceCurrentStatusData[mh.MonitorID][ss.serviceCurrentStatusIndex[mh.MonitorID]] = mh
		ss.serviceCurrentStatusIndex[mh.MonitorID]++
		// 数据持久化
		if ss.serviceCurrentStatusIndex[mh.MonitorID] == _CurrentStatusSize {
			ss.serviceCurrentStatusIndex[mh.MonitorID] = 0
			dataToSave := ss.serviceCurrentStatusData[mh.MonitorID]
			if err := DB.Create(&dataToSave).Error; err != nil {
				log.Println("NEZHA>> 服务监控数据持久化失败：", err)
			}
		}
		// 更新当前状态
		ss.serviceResponseDataStoreCurrentUp[mh.MonitorID] = 0
		ss.serviceResponseDataStoreCurrentDown[mh.MonitorID] = 0
		for i := 0; i < len(ss.serviceCurrentStatusData[mh.MonitorID]); i++ {
			if ss.serviceCurrentStatusData[mh.MonitorID][i].MonitorID > 0 {
				if ss.serviceCurrentStatusData[mh.MonitorID][i].Successful {
					ss.serviceResponseDataStoreCurrentUp[mh.MonitorID]++
				} else {
					ss.serviceResponseDataStoreCurrentDown[mh.MonitorID]++
				}
			}
		}
		var upPercent uint64 = 0
		if ss.serviceResponseDataStoreCurrentDown[mh.MonitorID]+ss.serviceResponseDataStoreCurrentUp[mh.MonitorID] > 0 {
			upPercent = ss.serviceResponseDataStoreCurrentUp[mh.MonitorID] * 100 / (ss.serviceResponseDataStoreCurrentDown[mh.MonitorID] + ss.serviceResponseDataStoreCurrentUp[mh.MonitorID])
		}
		stateStr := getStateStr(upPercent)
		if !mh.Successful {
			ServerLock.RLock()
			log.Println("NEZHA>> 服务故障上报：", ss.monitors[mh.MonitorID].Target, stateStr, "上报者：", ServerList[r.Reporter].Name, "请求输出：", mh.Data)
			ServerLock.RUnlock()
		}
		if stateStr == "故障" || stateStr != ss.lastStatus[mh.MonitorID] {
			ss.monitorsLock.RLock()
			isNeedSendNotification := (ss.lastStatus[mh.MonitorID] != "" || stateStr == "故障") && ss.monitors[mh.MonitorID].Notify
			ss.lastStatus[mh.MonitorID] = stateStr
			if isNeedSendNotification {
				go SendNotification(fmt.Sprintf("[服务%s] %s", stateStr, ss.monitors[mh.MonitorID].Name), true)
			}
			ss.monitorsLock.RUnlock()
		}
		ss.serviceResponseDataStoreLock.Unlock()
		// SSL 证书报警
		var errMsg string
		if strings.HasPrefix(mh.Data, "SSL证书错误：") {
			// 排除 i/o timeont、connection timeout、EOF 错误
			if !strings.HasSuffix(mh.Data, "timeout") &&
				!strings.HasSuffix(mh.Data, "EOF") &&
				!strings.HasSuffix(mh.Data, "timed out") {
				errMsg = mh.Data
			}
		} else {
			var newCert = strings.Split(mh.Data, "|")
			if len(newCert) > 1 {
				if ss.sslCertCache[mh.MonitorID] == "" {
					ss.sslCertCache[mh.MonitorID] = mh.Data
				}
				expiresNew, _ := time.Parse("2006-01-02 15:04:05 -0700 MST", newCert[1])
				// 证书过期提醒
				if expiresNew.Before(time.Now().AddDate(0, 0, 7)) {
					errMsg = fmt.Sprintf(
						"SSL证书将在七天内过期，过期时间：%s。",
						expiresNew.Format("2006-01-02 15:04:05"))
				}
				// 证书变更提醒
				var oldCert = strings.Split(ss.sslCertCache[mh.MonitorID], "|")
				var expiresOld time.Time
				if len(oldCert) > 1 {
					expiresOld, _ = time.Parse("2006-01-02 15:04:05 -0700 MST", oldCert[1])
				}
				if oldCert[0] != newCert[0] && !expiresNew.Equal(expiresOld) {
					ss.sslCertCache[mh.MonitorID] = mh.Data
					errMsg = fmt.Sprintf(
						"SSL证书变更，旧：%s, %s 过期；新：%s, %s 过期。",
						oldCert[0], expiresOld.Format("2006-01-02 15:04:05"), newCert[0], expiresNew.Format("2006-01-02 15:04:05"))
				}
			}
		}
		if errMsg != "" {
			ss.monitorsLock.RLock()
			if ss.monitors[mh.MonitorID].Notify {
				go SendNotification(fmt.Sprintf("[SSL] %s %s", ss.monitors[mh.MonitorID].Name, errMsg), true)
			}
			ss.monitorsLock.RUnlock()
		}
	}
}
