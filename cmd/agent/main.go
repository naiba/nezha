package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	pb "github.com/p14yground/nezha/proto"
	"github.com/p14yground/nezha/service/monitor"
	"github.com/p14yground/nezha/service/rpc"
)

var (
	rootCmd = &cobra.Command{
		Use:   "nezha-agent",
		Short: "「哪吒面板」监控、备份、站点管理一站式服务",
		Long: `哪吒面板
================================
监控、备份、站点管理一站式服务
啦啦啦，啦啦啦，我是 mjj 小行家`,
		Run: run,
	}
	appKey    string
	appSecret string
)

func main() {
	rootCmd.PersistentFlags().StringVarP(&appKey, "id", "i", "", "客户端ID")
	rootCmd.PersistentFlags().StringVarP(&appSecret, "secret", "p", "", "客户端Secret")
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	auth := rpc.AuthHandler{
		AppKey:    appKey,
		AppSecret: appSecret,
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
