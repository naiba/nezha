package singleton

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/jinzhu/copier"

	"github.com/robfig/cron/v3"

	"github.com/naiba/nezha/model"
	pb "github.com/naiba/nezha/proto"
)

var (
	Cron     *cron.Cron
	Crons    map[uint64]*model.Cron // [CrondID] -> *model.Cron
	CronLock sync.RWMutex
)

func InitCronTask() {
	Cron = cron.New(cron.WithSeconds(), cron.WithLocation(Loc))
	Crons = make(map[uint64]*model.Cron)
}

// loadCronTasks 加载计划任务
func loadCronTasks() {
	InitCronTask()
	var crons []model.Cron
	DB.Find(&crons)
	var err error
	var notificationTagList []string
	notificationMsgMap := make(map[string]*bytes.Buffer)
	for i := 0; i < len(crons); i++ {
		// 触发任务类型无需注册
		if crons[i].TaskType == model.CronTypeTriggerTask {
			Crons[crons[i].ID] = &crons[i]
			continue
		}
		// 旧版本计划任务可能不存在通知组 为其添加默认通知组
		if crons[i].NotificationTag == "" {
			crons[i].NotificationTag = "default"
			DB.Save(crons[i])
		}
		// 注册计划任务
		crons[i].CronJobID, err = Cron.AddFunc(crons[i].Scheduler, CronTrigger(crons[i]))
		if err == nil {
			Crons[crons[i].ID] = &crons[i]
		} else {
			// 当前通知组首次出现 将其加入通知组列表并初始化通知组消息缓存
			if _, ok := notificationMsgMap[crons[i].NotificationTag]; !ok {
				notificationTagList = append(notificationTagList, crons[i].NotificationTag)
				notificationMsgMap[crons[i].NotificationTag] = bytes.NewBufferString("")
				notificationMsgMap[crons[i].NotificationTag].WriteString("调度失败的计划任务：[")
			}
			notificationMsgMap[crons[i].NotificationTag].WriteString(fmt.Sprintf("%d,", crons[i].ID))
		}
	}
	// 向注册错误的计划任务所在通知组发送通知
	for _, tag := range notificationTagList {
		notificationMsgMap[tag].WriteString("] 这些任务将无法正常执行,请进入后点重新修改保存。")
		SendNotification(tag, notificationMsgMap[tag].String(), nil)
	}
	Cron.Start()
}

func ManualTrigger(c model.Cron) {
	CronTrigger(c)()
}

func SendTriggerTasks(taskIDs []uint64, triggerServer uint64) {
	CronLock.RLock()
	var cronLists []*model.Cron
	for _, taskID := range taskIDs {
		if c, ok := Crons[taskID]; ok {
			cronLists = append(cronLists, c)
		}
	}
	CronLock.RUnlock()

	// 依次调用CronTrigger发送任务
	for _, c := range cronLists {
		go CronTrigger(*c, triggerServer)()
	}
}

func CronTrigger(cr model.Cron, triggerServer ...uint64) func() {
	crIgnoreMap := make(map[uint64]bool)
	for j := 0; j < len(cr.Servers); j++ {
		crIgnoreMap[cr.Servers[j]] = true
	}
	return func() {
		if cr.Cover == model.CronCoverAlertTrigger {
			if len(triggerServer) == 0 {
				return
			}
			ServerLock.RLock()
			defer ServerLock.RUnlock()
			if s, ok := ServerList[triggerServer[0]]; ok {
				if s.TaskStream != nil {
					s.TaskStream.Send(&pb.Task{
						Id:   cr.ID,
						Data: cr.Command,
						Type: model.TaskTypeCommand,
					})
				} else {
					// 保存当前服务器状态信息
					curServer := model.Server{}
					copier.Copy(&curServer, s)
					SendNotification(cr.NotificationTag, fmt.Sprintf("[任务失败] %s，服务器 %s 离线，无法执行。", cr.Name, s.Name), nil, &curServer)
				}
			}
			return
		}

		ServerLock.RLock()
		defer ServerLock.RUnlock()
		for _, s := range ServerList {
			if cr.Cover == model.CronCoverAll && crIgnoreMap[s.ID] {
				continue
			}
			if cr.Cover == model.CronCoverIgnoreAll && !crIgnoreMap[s.ID] {
				continue
			}
			if s.TaskStream != nil {
				s.TaskStream.Send(&pb.Task{
					Id:   cr.ID,
					Data: cr.Command,
					Type: model.TaskTypeCommand,
				})
			} else {
				// 保存当前服务器状态信息
				curServer := model.Server{}
				copier.Copy(&curServer, s)
				SendNotification(cr.NotificationTag, fmt.Sprintf("[任务失败] %s，服务器 %s 离线，无法执行。", cr.Name, s.Name), nil, &curServer)
			}
		}
	}
}
