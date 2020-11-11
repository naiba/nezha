package rpc

import (
	"fmt"
	"net"

	"google.golang.org/grpc"

	pb "github.com/naiba/nezha/proto"
	rpcService "github.com/naiba/nezha/service/rpc"
)

// ServeRPC ...
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
