package main

import (
	"context"
	"net"

	pb "github.com/EDDYCJY/go-grpc-example/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// SearchService ..
type SearchService struct {
	auth *Auth
}

// Search ..
func (s *SearchService) Search(ctx context.Context, r *pb.SearchRequest) (*pb.SearchResponse, error) {
	if err := s.auth.Check(ctx); err != nil {
		return nil, err
	}
	return &pb.SearchResponse{Response: r.GetRequest() + " Token Server"}, nil
}

func main() {
	server := grpc.NewServer()
	pb.RegisterSearchServiceServer(server, &SearchService{})

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
