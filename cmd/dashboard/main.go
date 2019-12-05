package main

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pb "github.com/p14yground/nezha/proto"
)

// NezhaService ..
type NezhaService struct {
	auth *Auth
}

// ReportState ..
func (s *NezhaService) ReportState(ctx context.Context, r *pb.State) (*pb.Receipt, error) {
	if err := s.auth.Check(ctx); err != nil {
		return nil, err
	}
	fmt.Printf("receive: %s\n", r)
	return &pb.Receipt{}, nil
}

func main() {
	server := grpc.NewServer()
	pb.RegisterNezhaServiceServer(server, &NezhaService{})

	lis, err := net.Listen("tcp", ":5555")
	if err != nil {
		panic(err)
	}

	server.Serve(lis)
}

// Auth ..
type Auth struct {
	appKey    string
	appSecret string
}

// Check ..
func (a *Auth) Check(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Errorf(codes.Unauthenticated, "metadata.FromIncomingContext err")
	}

	var (
		appKey    string
		appSecret string
	)
	if value, ok := md["app_key"]; ok {
		appKey = value[0]
	}
	if value, ok := md["app_secret"]; ok {
		appSecret = value[0]
	}

	if appKey != a.GetAppKey() || appSecret != a.GetAppSecret() {
		return status.Errorf(codes.Unauthenticated, "invalid token")
	}

	return nil
}

// GetAppKey ..
func (a *Auth) GetAppKey() string {
	return "naiba"
}

// GetAppSecret ..
func (a *Auth) GetAppSecret() string {
	return "nbsecret"
}
