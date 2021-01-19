package rpc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/naiba/nezha/model"
	pb "github.com/naiba/nezha/proto"
	"github.com/naiba/nezha/service/alertmanager"
	"github.com/naiba/nezha/service/dao"
)

type NezhaHandler struct {
	Auth *AuthHandler
}

func (s *NezhaHandler) ReportTask(c context.Context, r *pb.TaskResult) (*pb.Receipt, error) {
	var err error
	var clientID uint64
	if clientID, err = s.Auth.Check(c); err != nil {
		return nil, err
	}
	if r.GetType() == model.TaskTypeHTTPGET {
		// SSL 证书报警
		var errMsg string
		if strings.HasPrefix(r.GetData(), "SSL证书错误：") {
			// 证书错误提醒
			errMsg = r.GetData()
		} else {
			var last model.MonitorHistory
			if err := dao.DB.Where("monitor_id = ? AND data LIKE ?", r.GetId(), "%|%").Order("id DESC").First(&last).Error; err == nil {
				var oldCert = strings.Split(last.Data, "|")
				var newCert = strings.Split(r.GetData(), "|")
				expiresOld, _ := time.Parse("2006-01-02 15:04:05 -0700 MST", oldCert[1])
				expiresNew, _ := time.Parse("2006-01-02 15:04:05 -0700 MST", newCert[1])
				// 证书变更提醒
				if last.Data != "" && oldCert[0] != newCert[0] && !expiresNew.Equal(expiresOld) {
					errMsg = fmt.Sprintf(
						"SSL证书变更，旧：%s, %s 过期；新：%s, %s 过期。",
						oldCert[0], expiresOld.Format("2006-01-02 15:04:05"), newCert[0], expiresNew.Format("2006-01-02 15:04:05"))
				}
				// 证书过期提醒
				if err == nil && expiresNew.Before(time.Now().AddDate(0, 0, 7)) {
					errMsg = fmt.Sprintf(
						"SSL证书将在七天内过期，过期时间：%s。",
						expiresNew.Format("2006-01-02 15:04:05"))
				}
			}
		}
		if errMsg != "" {
			var monitor model.Monitor
			dao.DB.First(&monitor, "id = ?", r.GetId())
			alertmanager.SendNotification(fmt.Sprintf("服务监控：%s %s", monitor.Name, errMsg))
		}
	}
	if r.GetType() == model.TaskTypeCommand {
		// 处理上报的计划任务
		dao.CronLock.RLock()
		cr := dao.Crons[r.GetId()]
		dao.CronLock.RUnlock()
		if cr.PushSuccessful && r.GetSuccessful() {
			alertmanager.SendNotification(fmt.Sprintf("成功计划任务：%s ，服务器：%d，日志：\n%s", cr.Name, clientID, r.GetData()))
		}
		if !r.GetSuccessful() {
			alertmanager.SendNotification(fmt.Sprintf("失败计划任务：%s ，服务器：%d，日志：\n%s", cr.Name, clientID, r.GetData()))
		}
		dao.DB.Model(cr).Updates(model.Cron{
			LastExecutedAt: time.Now().Add(time.Second * -1 * time.Duration(r.GetDelay())),
			LastResult:     r.GetSuccessful(),
		})
	} else {
		// 存入历史记录
		mh := model.PB2MonitorHistory(r)
		if err := dao.DB.Create(&mh).Error; err != nil {
			return nil, err
		}
	}
	return &pb.Receipt{Proced: true}, nil
}

func (s *NezhaHandler) RequestTask(h *pb.Host, stream pb.NezhaService_RequestTaskServer) error {
	var clientID uint64
	var err error
	if clientID, err = s.Auth.Check(stream.Context()); err != nil {
		return err
	}
	closeCh := make(chan error)
	dao.ServerLock.RLock()
	dao.ServerList[clientID].TaskStream = stream
	dao.ServerList[clientID].TaskClose = closeCh
	dao.ServerLock.RUnlock()
	select {
	case err = <-closeCh:
		return err
	}
}

func (s *NezhaHandler) ReportSystemState(c context.Context, r *pb.State) (*pb.Receipt, error) {
	var clientID uint64
	var err error
	if clientID, err = s.Auth.Check(c); err != nil {
		return nil, err
	}
	state := model.PB2State(r)
	dao.ServerLock.RLock()
	defer dao.ServerLock.RUnlock()
	dao.ServerList[clientID].LastActive = time.Now()
	dao.ServerList[clientID].State = &state
	return &pb.Receipt{Proced: true}, nil
}

func (s *NezhaHandler) ReportSystemInfo(c context.Context, r *pb.Host) (*pb.Receipt, error) {
	var clientID uint64
	var err error
	if clientID, err = s.Auth.Check(c); err != nil {
		return nil, err
	}
	host := model.PB2Host(r)
	dao.ServerLock.RLock()
	defer dao.ServerLock.RUnlock()
	if dao.Conf.EnableIPChangeNotification &&
		dao.ServerList[clientID].Host != nil &&
		dao.ServerList[clientID].Host.IP != "" &&
		host.IP != "" &&
		dao.ServerList[clientID].Host.IP != host.IP {
		alertmanager.SendNotification(fmt.Sprintf(
			"IP变更提醒 服务器：%s ，旧IP：%s，新IP：%s。",
			dao.ServerList[clientID].Name, dao.ServerList[clientID].Host.IP, host.IP))
	}
	dao.ServerList[clientID].Host = &host
	return &pb.Receipt{Proced: true}, nil
}
