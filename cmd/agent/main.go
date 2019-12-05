package main

import (
	"context"
	"log"

	"google.golang.org/grpc"

	pb "github.com/p14yground/nezha/proto"
)

// Auth ..
type Auth struct {
	AppKey    string
	AppSecret string
}

// GetRequestMetadata ..
func (a *Auth) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{"app_key": a.AppKey, "app_secret": a.AppSecret}, nil
}

// RequireTransportSecurity ..
func (a *Auth) RequireTransportSecurity() bool {
	return false
}

func main() {
	auth := Auth{
		AppKey:    "naiba",
		AppSecret: "nbsecret",
	}
	conn, err := grpc.Dial(":5555", grpc.WithInsecure(), grpc.WithPerRPCCredentials(&auth))
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	client := pb.NewNezhaServiceClient(conn)

	for i := 0; i < 3; i++ {
		resp, err := client.ReportState(context.Background(), &pb.State{})
		if err != nil {
			log.Fatalf("client.Search err: %v", err)
		}

		log.Printf("resp: %s", resp)
	}
}
