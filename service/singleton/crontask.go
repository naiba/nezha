package singleton

import (
	"bytes"
	"fmt"
	"github.com/naiba/nezha/model"
	pb "github.com/naiba/nezha/proto"
	"github.com/robfig/cron/v3"
	"sync"
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

// LoadCronTasks 加载计划任务
func LoadCronTasks() {
	InitCronTask()
	var crons []model.Cron
	DB.Find(&crons)
	var err error
	errMsg := new(bytes.Buffer)
	var notificationTagList []string
	for _, cr := range crons {
		// 旧版本计划任务可能不存在通知组 为其添加默认通知组
		if cr.NotificationTag == "" {
			AddDefaultCronNotificationTag(&cr)
		}
		// 注册计划任务
		cr.CronJobID, err = Cron.AddFunc(cr.Scheduler, CronTrigger(cr))
		if err == nil {
			Crons[cr.ID] = &cr
		} else {
			if errMsg.Len() == 0 {
				errMsg.WriteString("调度失败的计划任务：[")
			}
			errMsg.WriteString(fmt.Sprintf("%d,", cr.ID))
			notificationTagList = append(notificationTagList, cr.NotificationTag)
		}
	}
	if errMsg.Len() > 0 {
		msg := errMsg.String() + "] 这些任务将无法正常执行,请进入后点重新修改保存。"
		for _, tag := range notificationTagList {
			// 向调度错误的计划任务所包含的所有通知组发送通知
			SendNotification(tag, msg, false)
		}
	}
	Cron.Start()
}

// AddDefaultCronNotificationTag 添加默认的计划任务通知组
func AddDefaultCronNotificationTag(c *model.Cron) {
	CronLock.Lock()
	defer CronLock.Unlock()

	if c.NotificationTag == "" {
		c.NotificationTag = "default"
	}
	DB.Save(c)
}

func ManualTrigger(c model.Cron) {
	CronTrigger(c)()
}

func CronTrigger(cr model.Cron) func() {
	crIgnoreMap := make(map[uint64]bool)
	for j := 0; j < len(cr.Servers); j++ {
		crIgnoreMap[cr.Servers[j]] = true
	}
	return func() {
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
				SendNotification(cr.NotificationTag, fmt.Sprintf("[任务失败] %s，服务器 %s 离线，无法执行。", cr.Name, s.Name), false)
			}
		}
	}
}
