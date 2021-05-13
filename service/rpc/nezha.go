package rpc

import (
	"context"
	"fmt"
	"time"

	"github.com/naiba/nezha/model"
	pb "github.com/naiba/nezha/proto"
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
	if r.GetType() != model.TaskTypeCommand {
		dao.ServiceSentinelShared.Dispatch(dao.ReportData{
			Data:     r,
			Reporter: clientID,
		})
	} else {
		// 处理上报的计划任务
		dao.CronLock.RLock()
		defer dao.CronLock.RUnlock()
		cr := dao.Crons[r.GetId()]
		if cr != nil {
			if cr.PushSuccessful && r.GetSuccessful() {
				dao.SendNotification(fmt.Sprintf("成功计划任务：%s ，服务器：%d，日志：\n%s", cr.Name, clientID, r.GetData()), false)
			}
			if !r.GetSuccessful() {
				dao.SendNotification(fmt.Sprintf("失败计划任务：%s ，服务器：%d，日志：\n%s", cr.Name, clientID, r.GetData()), false)
			}
			dao.DB.Model(cr).Updates(model.Cron{
				LastExecutedAt: time.Now().Add(time.Second * -1 * time.Duration(r.GetDelay())),
				LastResult:     r.GetSuccessful(),
			})
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
	return <-closeCh
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
		dao.Conf.IgnoredIPNotificationServerIDs[clientID] != struct{}{} &&
		dao.ServerList[clientID].Host != nil &&
		dao.ServerList[clientID].Host.IP != "" &&
		host.IP != "" &&
		dao.ServerList[clientID].Host.IP != host.IP {
		dao.SendNotification(fmt.Sprintf(
			"IP变更提醒 服务器：%s ，旧IP：%s，新IP：%s。",
			dao.ServerList[clientID].Name, dao.ServerList[clientID].Host.IP, host.IP), true)
	}
	dao.ServerList[clientID].Host = &host
	return &pb.Receipt{Proced: true}, nil
}
