package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"

	pb "github.com/p14yground/nezha/proto"
	"github.com/p14yground/nezha/service/monitor"
	"github.com/p14yground/nezha/service/rpc"
)

func main() {
	auth := rpc.AuthHandler{
		AppKey:    "naiba",
		AppSecret: "123456",
	}
	conn, err := grpc.Dial(":5555", grpc.WithInsecure(), grpc.WithPerRPCCredentials(&auth))
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	client := pb.NewNezhaServiceClient(conn)
	ctx := context.Background()

	resp, err := client.Register(ctx, monitor.GetHost().PB())
	if err != nil {
		log.Printf("client.Register err: %v", err)
	}
	log.Printf("Register resp: %s", resp)

	hc, err := client.Heartbeat(ctx, &pb.Beat{
		Timestamp: fmt.Sprintf("%v", time.Now()),
	})
	if err != nil {
		log.Printf("client.Register err: %v", err)
	}
	log.Printf("Register resp: %s", hc)

	for i := 0; i < 3; i++ {
		resp, err := client.ReportState(ctx, monitor.GetState(3).PB())
		if err != nil {
			log.Printf("client.ReportState err: %v", err)
		}
		log.Printf("ReportState resp: %s", resp)
	}
}
