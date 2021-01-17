package rpc

import (
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"

	"github.com/naiba/nezha/model"
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
		var tasks []model.Monitor
		var hasAliveAgent bool
		dao.DB.Find(&tasks)
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
			if dao.SortedServerList[index].TaskStream == nil {
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
