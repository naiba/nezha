package rpc

import (
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"

	pb "github.com/naiba/nezha/proto"
	"github.com/naiba/nezha/service/dao"
	rpcService "github.com/naiba/nezha/service/rpc"
)

func ServeRPC(port uint) {
	server := grpc.NewServer()
	pb.RegisterNezhaServiceServer(server, &rpcService.NezhaHandler{
		Auth: &rpcService.AuthHandler{},
	})
	listen, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(err)
	}
	server.Serve(listen)
}

func DispatchTask(duration time.Duration) {
	var index uint64 = 0
	for {
		var hasAliveAgent bool
		tasks := dao.ServiceSentinelShared.Monitors()
		dao.SortedServerLock.RLock()
		startedAt := time.Now()
		for i := 0; i < len(tasks); i++ {
			if index >= uint64(len(dao.SortedServerList)) {
				index = 0
				if !hasAliveAgent {
					break
				}
				hasAliveAgent = false
			}
			// 1. 如果此任务不可使用此服务器请求，跳过这个服务器（有些 IPv6 only 开了 NAT64 的机器请求 IPv4 总会出问题）
			// 2. 如果服务器不在线，跳过这个服务器
			if tasks[i].SkipServers[dao.SortedServerList[index].ID] || dao.SortedServerList[index].TaskStream == nil {
				i--
				index++
				continue
			}
			hasAliveAgent = true
			dao.SortedServerList[index].TaskStream.Send(tasks[i].PB())
			index++
		}
		dao.SortedServerLock.RUnlock()
		time.Sleep(time.Until(startedAt.Add(duration)))
	}
}
