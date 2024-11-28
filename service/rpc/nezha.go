package rpc

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/nezhahq/nezha/pkg/ddns"
	geoipx "github.com/nezhahq/nezha/pkg/geoip"
	"github.com/nezhahq/nezha/pkg/grpcx"

	"github.com/jinzhu/copier"

	"github.com/nezhahq/nezha/model"
	pb "github.com/nezhahq/nezha/proto"
	"github.com/nezhahq/nezha/service/singleton"
)

var _ pb.NezhaServiceServer = (*NezhaHandler)(nil)

var NezhaHandlerSingleton *NezhaHandler

type NezhaHandler struct {
	Auth          *authHandler
	ioStreams     map[string]*ioStreamContext
	ioStreamMutex *sync.RWMutex
}

func NewNezhaHandler() *NezhaHandler {
	return &NezhaHandler{
		Auth:          &authHandler{},
		ioStreamMutex: new(sync.RWMutex),
		ioStreams:     make(map[string]*ioStreamContext),
	}
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
				singleton.SendNotification(cr.NotificationGroupID, fmt.Sprintf("[%s] %s, %s\n%s", singleton.Localizer.T("Scheduled Task Executed Successfully"),
					cr.Name, singleton.ServerList[clientID].Name, r.GetData()), nil, &curServer)
			}
			if !r.GetSuccessful() {
				singleton.SendNotification(cr.NotificationGroupID, fmt.Sprintf("[%s] %s, %s\n%s", singleton.Localizer.T("Scheduled Task Executed Failed"),
					cr.Name, singleton.ServerList[clientID].Name, r.GetData()), nil, &curServer)
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
	singleton.ServerList[clientID].TaskCloseLock.Lock()
	// 修复不断的请求 task 但是没有 return 导致内存泄漏
	if singleton.ServerList[clientID].TaskClose != nil {
		close(singleton.ServerList[clientID].TaskClose)
	}
	singleton.ServerList[clientID].TaskStream = stream
	singleton.ServerList[clientID].TaskClose = closeCh
	singleton.ServerList[clientID].TaskCloseLock.Unlock()
	singleton.ServerLock.RUnlock()
	return <-closeCh
}

func (s *NezhaHandler) ReportSystemState(stream pb.NezhaService_ReportSystemStateServer) error {
	var err error
	var clientID uint64
	if clientID, err = s.Auth.Check(stream.Context()); err != nil {
		return err
	}
	var state *pb.State
	for {
		state, err = stream.Recv()
		if err != nil {
			log.Printf("NEZHA>> ReportSystemState eror: %v, clientID: %d\n", err, clientID)
			return nil
		}
		state := model.PB2State(state)

		singleton.ServerLock.RLock()
		singleton.ServerList[clientID].LastActive = time.Now()
		singleton.ServerList[clientID].State = &state
		// 应对 dashboard 重启的情况，如果从未记录过，先打点，等到小时时间点时入库
		if singleton.ServerList[clientID].PrevTransferInSnapshot == 0 || singleton.ServerList[clientID].PrevTransferOutSnapshot == 0 {
			singleton.ServerList[clientID].PrevTransferInSnapshot = int64(state.NetInTransfer)
			singleton.ServerList[clientID].PrevTransferOutSnapshot = int64(state.NetOutTransfer)
		}
		singleton.ServerLock.RUnlock()

		stream.Send(&pb.Receipt{Proced: true})
	}
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

	/**
	 * 这里的 singleton 中的数据都是关机前的旧数据
	 * 当 agent 重启时，bootTime 变大，agent 端会先上报 host 信息，然后上报 state 信息
	 * 这是可以借助上报顺序的空档，将停机前的流量统计数据标记下来，加到下一个小时的数据点上
	 */
	if singleton.ServerList[clientID].Host != nil && singleton.ServerList[clientID].Host.BootTime < host.BootTime {
		singleton.ServerList[clientID].PrevTransferInSnapshot = singleton.ServerList[clientID].PrevTransferInSnapshot - int64(singleton.ServerList[clientID].State.NetInTransfer)
		singleton.ServerList[clientID].PrevTransferOutSnapshot = singleton.ServerList[clientID].PrevTransferOutSnapshot - int64(singleton.ServerList[clientID].State.NetOutTransfer)
	}

	singleton.ServerList[clientID].Host = &host
	return &pb.Receipt{Proced: true}, nil
}

func (s *NezhaHandler) IOStream(stream pb.NezhaService_IOStreamServer) error {
	if _, err := s.Auth.Check(stream.Context()); err != nil {
		return err
	}
	id, err := stream.Recv()
	if err != nil {
		return err
	}
	if id == nil || len(id.Data) < 4 || (id.Data[0] != 0xff && id.Data[1] != 0x05 && id.Data[2] != 0xff && id.Data[3] == 0x05) {
		return fmt.Errorf("invalid stream id")
	}

	streamId := string(id.Data[4:])

	if _, err := s.GetStream(streamId); err != nil {
		return err
	}
	iw := grpcx.NewIOStreamWrapper(stream)
	if err := s.AgentConnected(streamId, iw); err != nil {
		return err
	}
	iw.Wait()
	return nil
}

func (s *NezhaHandler) ReportGeoIP(c context.Context, r *pb.GeoIP) (*pb.GeoIP, error) {
	var clientID uint64
	var err error
	if clientID, err = s.Auth.Check(c); err != nil {
		return nil, err
	}

	geoip := model.PB2GeoIP(r)
	joinedIP := geoip.IP.Join()
	use6 := r.GetUse6()

	singleton.ServerLock.RLock()
	// 检查并更新DDNS
	if singleton.ServerList[clientID].EnableDDNS && joinedIP != "" &&
		(singleton.ServerList[clientID].GeoIP == nil || singleton.ServerList[clientID].GeoIP.IP != geoip.IP) {
		ipv4 := geoip.IP.IPv4Addr
		ipv6 := geoip.IP.IPv6Addr
		providers, err := singleton.GetDDNSProvidersFromProfiles(singleton.ServerList[clientID].DDNSProfiles, &ddns.IP{Ipv4Addr: ipv4, Ipv6Addr: ipv6})
		if err == nil {
			for _, provider := range providers {
				go func(provider *ddns.Provider) {
					provider.UpdateDomain(context.Background())
				}(provider)
			}
		} else {
			log.Printf("NEZHA>> 获取DDNS配置时发生错误: %v", err)
		}
	}

	// 发送IP变动通知
	if singleton.ServerList[clientID].GeoIP != nil && singleton.Conf.EnableIPChangeNotification &&
		((singleton.Conf.Cover == model.ConfigCoverAll && !singleton.Conf.IgnoredIPNotificationServerIDs[clientID]) ||
			(singleton.Conf.Cover == model.ConfigCoverIgnoreAll && singleton.Conf.IgnoredIPNotificationServerIDs[clientID])) &&
		singleton.ServerList[clientID].GeoIP.IP.Join() != "" &&
		joinedIP != "" &&
		singleton.ServerList[clientID].GeoIP.IP != geoip.IP {

		singleton.SendNotification(singleton.Conf.IPChangeNotificationGroupID,
			fmt.Sprintf(
				"[%s] %s, %s => %s",
				singleton.Localizer.T("IP Changed"),
				singleton.ServerList[clientID].Name, singleton.IPDesensitize(singleton.ServerList[clientID].GeoIP.IP.Join()),
				singleton.IPDesensitize(joinedIP),
			),
			nil)
	}
	singleton.ServerLock.RUnlock()

	// 根据内置数据库查询 IP 地理位置
	var ip string
	if geoip.IP.IPv6Addr != "" && (use6 || geoip.IP.IPv4Addr == "") {
		ip = geoip.IP.IPv6Addr
	} else {
		ip = geoip.IP.IPv4Addr
	}

	netIP := net.ParseIP(ip)
	location, err := geoipx.Lookup(netIP)
	if err != nil {
		log.Printf("NEZHA>> geoip.Lookup: %v", err)
	}
	geoip.CountryCode = location

	// 将地区码写入到 Host
	singleton.ServerLock.Lock()
	defer singleton.ServerLock.Unlock()
	singleton.ServerList[clientID].GeoIP = &geoip

	return &pb.GeoIP{Ip: nil, CountryCode: location}, nil
}
