package main

import (
	"filehub/pkg/proto"
	"filehub/pkg/server"
	"fmt"
	"net"

	"google.golang.org/grpc"
)

func main() {
	address := ":9999"
	listen, err := net.Listen("tcp", address)
	if err != nil {
		panic(err)
	}

	s := grpc.NewServer()
	proto.RegisterFileManagerServer(s, server.NewServer())
	fmt.Println("listening at", address)
	if err := s.Serve(listen); err != nil {
		panic(err)
	}
}
