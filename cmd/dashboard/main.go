package main

import (
	"net"

	"google.golang.org/grpc"

	pb "github.com/p14yground/nezha/proto"
	"github.com/p14yground/nezha/service/handler"
)

func main() {
	server := grpc.NewServer()
	pb.RegisterNezhaServiceServer(server, &handler.NezhaHandler{
		Auth: &handler.AuthHandler{
			AppKey:    "naiba",
			AppSecret: "123456",
		},
	})

	lis, err := net.Listen("tcp", ":5555")
	if err != nil {
		panic(err)
	}

	server.Serve(lis)
}
