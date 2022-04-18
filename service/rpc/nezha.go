package rpc

import (
	"context"
	"fmt"
	"github.com/jinzhu/copier"
	"time"

	"github.com/naiba/nezha/model"
	pb "github.com/naiba/nezha/proto"
	"github.com/naiba/nezha/service/singleton"
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
	if r.GetType() == model.TaskTypeCommand {
		// 处理上报的计划任务
		singleton.CronLock.RLock()
		defer singleton.CronLock.RUnlock()
		cr := singleton.Crons[r.GetId()]
		if cr != nil {
			singleton.ServerLock.RLock()
			defer singleton.ServerLock.RUnlock()
			// 保存当前服务器状态信息
			curServer := model.Server{}
			copier.Copy(&curServer, singleton.ServerList[clientID])
			if cr.PushSuccessful && r.GetSuccessful() {
				singleton.SendNotification(cr.NotificationTag, fmt.Sprintf("[任务成功] %s ，服务器：%s，日志：\n%s", cr.Name, singleton.ServerList[clientID].Name, r.GetData()), false, &curServer)
			}
			if !r.GetSuccessful() {
				singleton.SendNotification(cr.NotificationTag, fmt.Sprintf("[任务失败] %s ，服务器：%s，日志：\n%s", cr.Name, singleton.ServerList[clientID].Name, r.GetData()), false, &curServer)
			}
			singleton.DB.Model(cr).Updates(model.Cron{
				LastExecutedAt: time.Now().Add(time.Second * -1 * time.Duration(r.GetDelay())),
				LastResult:     r.GetSuccessful(),
			})
		}
	} else if model.IsServiceSentinelNeeded(r.GetType()) {
		singleton.ServiceSentinelShared.Dispatch(singleton.ReportData{
			Data:     r,
			Reporter: clientID,
		})
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
	singleton.ServerLock.RLock()
	// 修复不断的请求 task 但是没有 return 导致内存泄漏
	if singleton.ServerList[clientID].TaskClose != nil {
		close(singleton.ServerList[clientID].TaskClose)
	}
	singleton.ServerList[clientID].TaskStream = stream
	singleton.ServerList[clientID].TaskClose = closeCh
	singleton.ServerLock.RUnlock()
	return <-closeCh
}

func (s *NezhaHandler) ReportSystemState(c context.Context, r *pb.State) (*pb.Receipt, error) {
	var clientID uint64
	var err error
	if clientID, err = s.Auth.Check(c); err != nil {
		return nil, err
	}
	state := model.PB2State(r)
	singleton.ServerLock.RLock()
	defer singleton.ServerLock.RUnlock()
	singleton.ServerList[clientID].LastActive = time.Now()
	singleton.ServerList[clientID].State = &state

	// 如果从未记录过，先打点，等到小时时间点时入库
	if singleton.ServerList[clientID].PrevHourlyTransferIn == 0 || singleton.ServerList[clientID].PrevHourlyTransferOut == 0 {
		singleton.ServerList[clientID].PrevHourlyTransferIn = int64(state.NetInTransfer)
		singleton.ServerList[clientID].PrevHourlyTransferOut = int64(state.NetOutTransfer)
	}

	return &pb.Receipt{Proced: true}, nil
}

func (s *NezhaHandler) ReportSystemInfo(c context.Context, r *pb.Host) (*pb.Receipt, error) {
	var clientID uint64
	var err error
	if clientID, err = s.Auth.Check(c); err != nil {
		return nil, err
	}
	host := model.PB2Host(r)
	singleton.ServerLock.RLock()
	defer singleton.ServerLock.RUnlock()
	if singleton.Conf.EnableIPChangeNotification &&
		((singleton.Conf.Cover == model.ConfigCoverAll && !singleton.Conf.IgnoredIPNotificationServerIDs[clientID]) ||
			(singleton.Conf.Cover == model.ConfigCoverIgnoreAll && singleton.Conf.IgnoredIPNotificationServerIDs[clientID])) &&
		singleton.ServerList[clientID].Host != nil &&
		singleton.ServerList[clientID].Host.IP != "" &&
		host.IP != "" &&
		singleton.ServerList[clientID].Host.IP != host.IP {
		singleton.SendNotification(singleton.Conf.IPChangeNotificationTag, fmt.Sprintf(
			"[IP变更] %s ，旧IP：%s，新IP：%s。",
			singleton.ServerList[clientID].Name, singleton.IPDesensitize(singleton.ServerList[clientID].Host.IP), singleton.IPDesensitize(host.IP)), true)
	}

	// 判断是否是机器重启，如果是机器重启要录入最后记录的流量里面
	if singleton.ServerList[clientID].Host.BootTime < host.BootTime {
		singleton.ServerList[clientID].PrevHourlyTransferIn = singleton.ServerList[clientID].PrevHourlyTransferIn - int64(singleton.ServerList[clientID].State.NetInTransfer)
		singleton.ServerList[clientID].PrevHourlyTransferOut = singleton.ServerList[clientID].PrevHourlyTransferOut - int64(singleton.ServerList[clientID].State.NetOutTransfer)
	}

	singleton.ServerList[clientID].Host = &host
	return &pb.Receipt{Proced: true}, nil
}
