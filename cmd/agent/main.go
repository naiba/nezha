package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/blang/semver"
	"github.com/p14yground/go-github-selfupdate/selfupdate"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"github.com/naiba/nezha/model"
	pb "github.com/naiba/nezha/proto"
	"github.com/naiba/nezha/service/dao"
	"github.com/naiba/nezha/service/monitor"
	"github.com/naiba/nezha/service/rpc"
)

var (
	clientID     string
	server       string
	clientSecret string
	debug        bool
	version      string

	rootCmd = &cobra.Command{
		Use:   "nezha-agent",
		Short: "「哪吒面板」监控、备份、站点管理一站式服务",
		Long: `哪吒面板
================================
监控、备份、站点管理一站式服务
啦啦啦，啦啦啦，我是 mjj 小行家`,
		Run:     run,
		Version: version,
	}
)

var (
	reporting      bool
	client         pb.NezhaServiceClient
	ctx            = context.Background()
	delayWhenError = time.Second * 10
	updateCh       = make(chan struct{}, 0)
)

func doSelfUpdate() {
	defer func() {
		time.Sleep(time.Minute * 20)
		updateCh <- struct{}{}
	}()
	v := semver.MustParse(version)
	log.Println("check update", v)
	latest, err := selfupdate.UpdateSelf(v, "naiba/nezha")
	if err != nil {
		log.Println("Binary update failed:", err)
		return
	}
	if latest.Version.Equals(v) {
		// latest version is the same as current version. It means current binary is up to date.
		log.Println("Current binary is the latest version", version)
	} else {
		log.Println("Successfully updated to version", latest.Version)
		os.Exit(1)
	}
}

func main() {
	// 来自于 GoReleaser 的版本号
	dao.Version = version

	rootCmd.PersistentFlags().StringVarP(&server, "server", "s", "localhost:5555", "客户端ID")
	rootCmd.PersistentFlags().StringVarP(&clientID, "id", "i", "", "客户端ID")
	rootCmd.PersistentFlags().StringVarP(&clientSecret, "secret", "p", "", "客户端Secret")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "开启Debug")
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	dao.Conf = &model.Config{
		Debug: debug,
	}
	auth := rpc.AuthHandler{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}

	// 上报服务器信息
	go reportState()

	if version != "" {
		go func() {
			for range updateCh {
				go doSelfUpdate()
			}
		}()
		updateCh <- struct{}{}
	}

	var err error
	var conn *grpc.ClientConn
	var hc pb.NezhaService_HeartbeatClient

	retry := func() {
		log.Println("Error to close connection ...")
		if conn != nil {
			conn.Close()
		}
		time.Sleep(delayWhenError)
		log.Println("Try to reconnect ...")
	}

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
		default:
			log.Printf("Unknown action: %v", action)
		}
	}
}

func reportState() {
	var err error
	defer log.Printf("reportState exit %v => %v", time.Now(), err)
	for {
		if client != nil {
			monitor.TrackNetworkSpeed()
			_, err = client.ReportState(ctx, monitor.GetState(dao.ReportDelay).PB())
			if err != nil {
				log.Printf("reportState error %v", err)
				time.Sleep(delayWhenError)
			}
		}
	}
}
