package rpc

import (
	"context"
	"log"

	"github.com/p14yground/nezha/model"
	pb "github.com/p14yground/nezha/proto"
	"github.com/p14yground/nezha/service/dao"
)

// NezhaHandler ..
type NezhaHandler struct {
	Auth *AuthHandler
}

// ReportState ..
func (s *NezhaHandler) ReportState(c context.Context, r *pb.State) (*pb.Receipt, error) {
	var clientID string
	var err error
	if clientID, err = s.Auth.Check(c); err != nil {
		return nil, err
	}
	dao.ServerLock.RLock()
	defer dao.ServerLock.RUnlock()
	dao.ServerList[clientID].State = model.PB2State(r)
	return &pb.Receipt{Proced: true}, nil
}

// Heartbeat ..
func (s *NezhaHandler) Heartbeat(r *pb.Beat, stream pb.NezhaService_HeartbeatServer) error {
	var clientID string
	var err error
	defer log.Printf("Heartbeat exit server:%v err:%v", clientID, err)
	if clientID, err = s.Auth.Check(stream.Context()); err != nil {
		return err
	}
	dao.ServerLock.RLock()
	defer dao.ServerLock.RUnlock()
	closeCh := make(chan error)
	dao.ServerList[clientID].StreamClose = closeCh
	dao.ServerList[clientID].Stream = stream
	select {
	case err = <-closeCh:
		return err
	}
}

// Register ..
func (s *NezhaHandler) Register(c context.Context, r *pb.Host) (*pb.Receipt, error) {
	var clientID string
	var err error
	if clientID, err = s.Auth.Check(c); err != nil {
		return nil, err
	}
	dao.ServerLock.RLock()
	defer dao.ServerLock.RUnlock()
	dao.ServerList[clientID].Host = model.PB2Host(r)
	return &pb.Receipt{Proced: true}, nil
}
