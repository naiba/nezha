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
	dao.ServerLock.Lock()
	defer dao.ServerLock.Unlock()
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
	err = stream.Send(&pb.Command{
		Type: model.MTReportState,
	})
	if err != nil {
		log.Printf("Heartbeat stream.Send err:%v", err)
	}
	select {}
}

// Register ..
func (s *NezhaHandler) Register(c context.Context, r *pb.Host) (*pb.Receipt, error) {
	var clientID string
	var err error
	if clientID, err = s.Auth.Check(c); err != nil {
		return nil, err
	}
	dao.ServerLock.Lock()
	defer dao.ServerLock.Unlock()
	dao.ServerList[clientID].Host = model.PB2Host(r)
	return &pb.Receipt{Proced: true}, nil
}
