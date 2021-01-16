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
	if _, err = s.Auth.Check(c); err != nil {
		return nil, err
	}
	if r.GetType() == model.MonitorTypeHTTPGET {
		// SSL 证书报警
		var last model.MonitorHistory
		if err := dao.DB.Where("monitor_id = ?", r.GetId()).Order("id DESC").First(&last).Error; err == nil {
			var errMsg string
			if strings.HasPrefix(r.GetData(), "SSL证书错误：") {
				// 证书错误提醒
				errMsg = r.GetData()
			} else {
				var oldSSLCert = strings.Split(last.Data, "|")
				var splits = strings.Split(r.GetData(), "|")
				// 证书变更提醒
				if last.Data != "" && oldSSLCert[0] != splits[0] {
					errMsg = fmt.Sprintf(
						"SSL证书变更，旧：%s，新：%s。",
						last.Data, splits[0])
				}
				expires, err := time.Parse("2006-01-02 15:04:05 -0700 MST", splits[1])
				// 证书过期提醒
				if err == nil && expires.Before(time.Now().AddDate(0, 0, 7)) {
					errMsg = fmt.Sprintf(
						"SSL证书将在七天内过期，过期时间：%s。",
						expires.Format("2006-01-02 15:04:05"))
				}
			}
			if errMsg != "" {
				var monitor model.Monitor
				dao.DB.First(&monitor, "id = ?", last.MonitorID)
				alertmanager.SendNotification(fmt.Sprintf("服务监控：%s %s", monitor.Name, errMsg))
			}
		}
	}
	// 存入历史记录
	mh := model.PB2MonitorHistory(r)
	if err := dao.DB.Create(&mh).Error; err != nil {
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
			"IP变更提醒 服务器：%s ，旧IP：%s，新IP：%s。",
			dao.ServerList[clientID].Name, dao.ServerList[clientID].Host.IP, host.IP))
	}
	dao.ServerList[clientID].Host = &host
	return &pb.Receipt{Proced: true}, nil
}
