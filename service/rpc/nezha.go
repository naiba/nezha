package rpc

import (
	"context"
	"fmt"
	"log"

	"github.com/p14yground/nezha/model"
	pb "github.com/p14yground/nezha/proto"
)

// NezhaHandler ..
type NezhaHandler struct {
	Auth *AuthHandler
}

// ReportState ..
func (s *NezhaHandler) ReportState(c context.Context, r *pb.State) (*pb.Receipt, error) {
	if err := s.Auth.Check(c); err != nil {
		return nil, err
	}
	fmt.Printf("ReportState receive: %s\n", r)
	return &pb.Receipt{Proced: true}, nil
}

// Heartbeat ..
func (s *NezhaHandler) Heartbeat(r *pb.Beat, stream pb.NezhaService_HeartbeatServer) error {
	defer log.Println("Heartbeat exit")
	if err := s.Auth.Check(stream.Context()); err != nil {
		return err
	}
	err := stream.Send(&pb.Command{
		Type: model.MTReportState,
	})
	if err != nil {
		log.Printf("Heartbeat stream.Send err:%v", err)
	}
	select {}
}

// Register ..
func (s *NezhaHandler) Register(c context.Context, r *pb.Host) (*pb.Receipt, error) {
	if err := s.Auth.Check(c); err != nil {
		return nil, err
	}
	fmt.Printf("Register receive: %s\n", r)
	return &pb.Receipt{Proced: true}, nil
}
