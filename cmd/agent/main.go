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
	server       string
	clientSecret string
	debug        bool
)

func main() {
	rootCmd.PersistentFlags().StringVarP(&server, "server", "s", "localhost:5555", "客户端ID")
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

func run(cmd *cobra.Command, args []string) {
	dao.Conf = &model.Config{
		Debug: debug,
	}
	auth := rpc.AuthHandler{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}
	retry := func() {
		time.Sleep(delayWhenError)
		log.Println("Try to reconnect ...")
	}

	// 上报服务器信息
	go reportState()

	var err error
	var conn *grpc.ClientConn
	var hc pb.NezhaService_HeartbeatClient

	for {
		conn, err = grpc.Dial(server, grpc.WithInsecure(), grpc.WithPerRPCCredentials(&auth))
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
		// 心跳接收控制命令
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
			monitor.TrackNetworkSpeed()
			_, err = client.ReportState(ctx, monitor.GetState(2).PB())
			if err != nil {
				log.Printf("reportState error %v", err)
				time.Sleep(delayWhenError)
			}
		} else {
			time.Sleep(time.Second * 1)
		}
	}
}
