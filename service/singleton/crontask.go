package singleton

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/robfig/cron/v3"

	"github.com/naiba/nezha/model"
	pb "github.com/naiba/nezha/proto"
)

var (
	Cron     *cron.Cron
	Crons    map[uint64]*model.Cron
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
	for i := 0; i < len(crons); i++ {
		cr := crons[i]

		// 注册计划任务
		cr.CronJobID, err = Cron.AddFunc(cr.Scheduler, CronTrigger(cr))
		if err == nil {
			Crons[cr.ID] = &cr
		} else {
			if errMsg.Len() == 0 {
				errMsg.WriteString("调度失败的计划任务：[")
			}
			errMsg.WriteString(fmt.Sprintf("%d,", cr.ID))
		}
	}
	if errMsg.Len() > 0 {
		msg := errMsg.String()
		SendNotification(msg[:len(msg)-1]+"] 这些任务将无法正常执行,请进入后点重新修改保存。", false)
	}
	Cron.Start()
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
				SendNotification(fmt.Sprintf("[任务失败] %s，服务器 %s 离线，无法执行。", cr.Name, s.Name), false)
			}
		}
	}
}
