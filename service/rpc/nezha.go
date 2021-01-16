package rpc

import (
	"context"
	"fmt"
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
	if _, err = s.Auth.Check(c); err != nil {
		return nil, err
	}
	if r.GetType() == model.MonitorTypeHTTPGET {
		// SSL 证书变更报警
		var last model.MonitorHistory
		if err := dao.DB.Where("monitor_id = ?", r.GetId()).Order("id DESC").First(&last).Error; err == nil {
			if last.Data != "" && last.Data != r.GetData() {
				var monitor model.Monitor
				dao.DB.First(&monitor, "id = ?", last.MonitorID)
				alertmanager.SendNotification(fmt.Sprintf(
					"监控：%s SSL证书变更，旧：%s，新：%s。",
					monitor.Name, last.Data, r.GetData()))
			}
		}
	}
	// 存入历史记录
	mh := model.PB2MonitorHistory(r)
	if err := dao.DB.Create(&mh).Error; err != nil {
		return nil, err
	}
	// 更新最后检测时间
	var m model.Monitor
	m.ID = r.GetId()
	if err := dao.DB.Model(&m).Update("last_check", time.Now()).Error; err != nil {
		return nil, err
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
	dao.ServerLock.Lock()
	dao.ServerList[clientID].TaskStream = stream
	dao.ServerList[clientID].TaskClose = closeCh
	dao.ServerLock.Unlock()
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
			"服务器：%s IP变更提醒，旧IP：%s，新IP：%s。",
			dao.ServerList[clientID].Name, dao.ServerList[clientID].Host.IP, host.IP))
	}
	dao.ServerList[clientID].Host = &host
	return &pb.Receipt{Proced: true}, nil
}
