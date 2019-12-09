package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"github.com/p14yground/nezha/model"
	pb "github.com/p14yground/nezha/proto"
	"github.com/p14yground/nezha/service/dao"
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
	clientID     string
	clientSecret string
	debug        bool
)

func main() {
	rootCmd.PersistentFlags().StringVarP(&clientID, "id", "i", "", "客户端ID")
	rootCmd.PersistentFlags().StringVarP(&clientSecret, "secret", "p", "", "客户端Secret")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "开启Debug")
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var endReport time.Time
var reporting bool
var client pb.NezhaServiceClient
var ctx = context.Background()
var delayWhenError = time.Second * 10
var delayWhenReport = time.Second

func run(cmd *cobra.Command, args []string) {
	dao.Conf = &model.Config{
		Debug: debug,
	}
	auth := rpc.AuthHandler{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}
	go reportState()
	var err error
	var conn *grpc.ClientConn
	var hc pb.NezhaService_HeartbeatClient
	retry := func() {
		time.Sleep(delayWhenError)
		log.Println("Try to reconnect ...")
	}
	for {
		conn, err = grpc.Dial(":5555", grpc.WithInsecure(), grpc.WithPerRPCCredentials(&auth))
		if err != nil {
			log.Printf("grpc.Dial err: %v", err)
			retry()
			continue
		}
		client = pb.NewNezhaServiceClient(conn)
		// 第一步注册
		_, err = client.Register(ctx, monitor.GetHost().PB())
		if err != nil {
			log.Printf("client.Register err: %v", err)
			retry()
			continue
		}
		hc, err = client.Heartbeat(ctx, &pb.Beat{
			Timestamp: fmt.Sprintf("%v", time.Now()),
		})
		if err != nil {
			log.Printf("client.Heartbeat err: %v", err)
			retry()
			continue
		}
		err = receiveCommand(hc)
		log.Printf("receiveCommand exit to main: %v", err)
		retry()
	}
}

func receiveCommand(hc pb.NezhaService_HeartbeatClient) error {
	var err error
	var action *pb.Command
	defer log.Printf("receiveCommand exit %v %v => %v", time.Now(), action, err)
	for {
		action, err = hc.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		switch action.GetType() {
		case model.MTReportState:
			endReport = time.Now().Add(time.Minute * 10)
		default:
			log.Printf("Unknown action: %v", action)
		}
	}
}

func reportState() {
	var err error
	defer log.Printf("reportState exit %v %v => %v", endReport, time.Now(), err)
	for {
		if endReport.After(time.Now()) {
			_, err = client.ReportState(ctx, monitor.GetState(0).PB())
			if err != nil {
				log.Printf("reportState error %v", err)
				time.Sleep(delayWhenError)
			}
		}
		time.Sleep(delayWhenReport)
	}
}
