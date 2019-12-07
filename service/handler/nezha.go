package handler

import (
	"context"
	"fmt"

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
	if err := s.Auth.Check(stream.Context()); err != nil {
		return err
	}
	fmt.Printf("ReportState receive: %s\n", r)
	return nil
}

// Register ..
func (s *NezhaHandler) Register(c context.Context, r *pb.Host) (*pb.Receipt, error) {
	if err := s.Auth.Check(c); err != nil {
		return nil, err
	}
	fmt.Printf("Register receive: %s\n", r)
	return &pb.Receipt{Proced: true}, nil
}
