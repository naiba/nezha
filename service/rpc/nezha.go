package rpc

import (
	"context"
	"log"
	"time"

	"github.com/naiba/nezha/model"
	pb "github.com/naiba/nezha/proto"
	"github.com/naiba/nezha/service/dao"
)

// NezhaHandler ..
type NezhaHandler struct {
	Auth *AuthHandler
}

// ReportState ..
func (s *NezhaHandler) ReportState(c context.Context, r *pb.State) (*pb.Receipt, error) {
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

// Heartbeat ..
func (s *NezhaHandler) Heartbeat(r *pb.Beat, stream pb.NezhaService_HeartbeatServer) error {
	var clientID uint64
	var err error
	defer log.Printf("Heartbeat exit server:%v err:%v", clientID, err)
	if clientID, err = s.Auth.Check(stream.Context()); err != nil {
		return err
	}
	// 放入在线服务器列表
	dao.ServerLock.RLock()
	closeCh := make(chan error)
	dao.ServerList[clientID].StreamClose = closeCh
	dao.ServerList[clientID].Stream = stream
	dao.ServerLock.RUnlock()
	select {
	case err = <-closeCh:
		return err
	}
}

// Register ..
func (s *NezhaHandler) Register(c context.Context, r *pb.Host) (*pb.Receipt, error) {
	var clientID uint64
	var err error
	if clientID, err = s.Auth.Check(c); err != nil {
		return nil, err
	}
	host := model.PB2Host(r)
	dao.ServerLock.RLock()
	defer dao.ServerLock.RUnlock()
	dao.ServerList[clientID].Host = &host
	return &pb.Receipt{Proced: true}, nil
}
