package rpc

import (
	"context"
	"fmt"
	"github.com/naiba/nezha/pkg/ddns"
	"github.com/naiba/nezha/pkg/utils"
	"log"
	"time"

	"github.com/jinzhu/copier"
	"github.com/nicksnyder/go-i18n/v2/i18n"

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
				singleton.SendNotification(cr.NotificationTag, fmt.Sprintf("[%s] %s, %s\n%s", singleton.Localizer.MustLocalize(
					&i18n.LocalizeConfig{
						MessageID: "ScheduledTaskExecutedSuccessfully",
					},
				), cr.Name, singleton.ServerList[clientID].Name, r.GetData()), nil, &curServer)
			}
			if !r.GetSuccessful() {
				singleton.SendNotification(cr.NotificationTag, fmt.Sprintf("[%s] %s, %s\n%s", singleton.Localizer.MustLocalize(
					&i18n.LocalizeConfig{
						MessageID: "ScheduledTaskExecutedFailed",
					},
				), cr.Name, singleton.ServerList[clientID].Name, r.GetData()), nil, &curServer)
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
	var provider ddns.Provider
	var err error
	if clientID, err = s.Auth.Check(c); err != nil {
		return nil, err
	}
	host := model.PB2Host(r)
	singleton.ServerLock.RLock()
	defer singleton.ServerLock.RUnlock()

	// 检查并更新DDNS
	if singleton.Conf.DDNS.Enable &&
		singleton.ServerList[clientID].EnableDDNS &&
		singleton.ServerList[clientID].Host != nil &&
		host.IP != "" &&
		singleton.ServerList[clientID].Host.IP != host.IP {

		serverDomain := singleton.ServerList[clientID].DDNSDomain
		if singleton.Conf.DDNS.Provider == "" {
			provider, err = singleton.GetDDNSProviderFromProfile(singleton.ServerList[clientID].DDNSProfile)
		} else {
			provider, err = singleton.GetDDNSProviderFromString(singleton.Conf.DDNS.Provider)
		}
		if err == nil && serverDomain != "" {
			ipv4, ipv6, _ := utils.SplitIPAddr(host.IP)
			maxRetries := int(singleton.Conf.DDNS.MaxRetries)
			config := &ddns.DomainConfig{
				EnableIPv4: singleton.ServerList[clientID].EnableIPv4,
				EnableIpv6: singleton.ServerList[clientID].EnableIpv6,
				FullDomain: serverDomain,
				Ipv4Addr:   ipv4,
				Ipv6Addr:   ipv6,
			}
			go singleton.RetryableUpdateDomain(provider, config, maxRetries)

		} else {
			// 虽然会在启动时panic, 可以断言不会走这个分支, 但是考虑到动态加载配置或者其它情况, 这里输出一下方便检查奇奇怪怪的BUG
			log.Printf("NEZHA>> 未找到对应的DDNS配置(%s), 或者是provider填写不正确, 请前往config.yml检查你的设置\n", singleton.ServerList[clientID].DDNSProfile)
		}

	}

	// 发送IP变动通知
	if singleton.Conf.EnableIPChangeNotification &&
		((singleton.Conf.Cover == model.ConfigCoverAll && !singleton.Conf.IgnoredIPNotificationServerIDs[clientID]) ||
			(singleton.Conf.Cover == model.ConfigCoverIgnoreAll && singleton.Conf.IgnoredIPNotificationServerIDs[clientID])) &&
		singleton.ServerList[clientID].Host != nil &&
		singleton.ServerList[clientID].Host.IP != "" &&
		host.IP != "" &&
		singleton.ServerList[clientID].Host.IP != host.IP {

		singleton.SendNotification(singleton.Conf.IPChangeNotificationTag,
			fmt.Sprintf(
				"[%s] %s, %s => %s",
				singleton.Localizer.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "IPChanged",
				}),
				singleton.ServerList[clientID].Name, singleton.IPDesensitize(singleton.ServerList[clientID].Host.IP),
				singleton.IPDesensitize(host.IP),
			),
			nil)
	}

	// 判断是否是机器重启，如果是机器重启要录入最后记录的流量里面
	if singleton.ServerList[clientID].Host.BootTime < host.BootTime {
		singleton.ServerList[clientID].PrevHourlyTransferIn = singleton.ServerList[clientID].PrevHourlyTransferIn - int64(singleton.ServerList[clientID].State.NetInTransfer)
		singleton.ServerList[clientID].PrevHourlyTransferOut = singleton.ServerList[clientID].PrevHourlyTransferOut - int64(singleton.ServerList[clientID].State.NetOutTransfer)
	}

	singleton.ServerList[clientID].Host = &host
	return &pb.Receipt{Proced: true}, nil
}
