package rpc

import (
	"fmt"
	"net"

	"google.golang.org/grpc"

	"github.com/naiba/nezha/model"
	pb "github.com/naiba/nezha/proto"
	rpcService "github.com/naiba/nezha/service/rpc"
	"github.com/naiba/nezha/service/singleton"
)

func ServeRPC(port uint) {
	server := grpc.NewServer()
	rpcService.NezhaHandlerSingleton = rpcService.NewNezhaHandler()
	pb.RegisterNezhaServiceServer(server, rpcService.NezhaHandlerSingleton)
	listen, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(err)
	}
	server.Serve(listen)
}

func DispatchTask(serviceSentinelDispatchBus <-chan model.Monitor) {
	workedServerIndex := 0
	for task := range serviceSentinelDispatchBus {
		round := 0
		endIndex := workedServerIndex
		singleton.SortedServerLock.RLock()
		// 如果已经轮了一整圈又轮到自己，没有合适机器去请求，跳出循环
		for round < 1 || workedServerIndex < endIndex {
			// 如果到了圈尾，再回到圈头，圈数加一，游标重置
			if workedServerIndex >= len(singleton.SortedServerList) {
				workedServerIndex = 0
				round++
				continue
			}
			// 如果服务器不在线，跳过这个服务器
			if singleton.SortedServerList[workedServerIndex].TaskStream == nil {
				workedServerIndex++
				continue
			}
			// 如果此任务不可使用此服务器请求，跳过这个服务器（有些 IPv6 only 开了 NAT64 的机器请求 IPv4 总会出问题）
			if (task.Cover == model.MonitorCoverAll && task.SkipServers[singleton.SortedServerList[workedServerIndex].ID]) ||
				(task.Cover == model.MonitorCoverIgnoreAll && !task.SkipServers[singleton.SortedServerList[workedServerIndex].ID]) {
				workedServerIndex++
				continue
			}
			if task.Cover == model.MonitorCoverIgnoreAll && task.SkipServers[singleton.SortedServerList[workedServerIndex].ID] {
				singleton.SortedServerList[workedServerIndex].TaskStream.Send(task.PB())
				workedServerIndex++
				continue
			}
			if task.Cover == model.MonitorCoverAll && !task.SkipServers[singleton.SortedServerList[workedServerIndex].ID] {
				singleton.SortedServerList[workedServerIndex].TaskStream.Send(task.PB())
				workedServerIndex++
				continue
			}
			// 找到合适机器执行任务，跳出循环
			// singleton.SortedServerList[workedServerIndex].TaskStream.Send(task.PB())
			// workedServerIndex++
			// break
		}
		singleton.SortedServerLock.RUnlock()
	}
}

func DispatchKeepalive() {
	singleton.Cron.AddFunc("@every 60s", func() {
		singleton.SortedServerLock.RLock()
		defer singleton.SortedServerLock.RUnlock()
		for i := 0; i < len(singleton.SortedServerList); i++ {
			if singleton.SortedServerList[i] == nil || singleton.SortedServerList[i].TaskStream == nil {
				continue
			}

			singleton.SortedServerList[i].TaskStream.Send(&pb.Task{Type: model.TaskTypeKeepalive})
		}
	})
}
