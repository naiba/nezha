package main

import (
	"context"
	"log"

	pb "github.com/EDDYCJY/go-grpc-example/proto"
	"google.golang.org/grpc"
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
		AppSecret: "nbsecret0",
	}
	conn, err := grpc.Dial(":5555", grpc.WithInsecure(), grpc.WithPerRPCCredentials(&auth))
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	client := pb.NewSearchServiceClient(conn)
	resp, err := client.Search(context.Background(), &pb.SearchRequest{
		Request: "gRPC",
	})
	if err != nil {
		log.Fatalf("client.Search err: %v", err)
	}

	log.Printf("resp: %s", resp.GetResponse())
}
