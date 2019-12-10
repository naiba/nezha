package rpc

import (
	"fmt"
	"net"

	"google.golang.org/grpc"

	pb "github.com/p14yground/nezha/proto"
	rpcService "github.com/p14yground/nezha/service/rpc"
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
