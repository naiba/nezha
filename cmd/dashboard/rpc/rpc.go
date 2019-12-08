package rpc

import (
	"net"

	"google.golang.org/grpc"

	pb "github.com/p14yground/nezha/proto"
	rpcService "github.com/p14yground/nezha/service/rpc"
)

// ServeRPC ...
func ServeRPC() {
	server := grpc.NewServer()
	pb.RegisterNezhaServiceServer(server, &rpcService.NezhaHandler{
		Auth: &rpcService.AuthHandler{
			AppKey:    "naiba",
			AppSecret: "123456",
		},
	})
	listen, err := net.Listen("tcp", ":5555")
	if err != nil {
		panic(err)
	}
	server.Serve(listen)
}
