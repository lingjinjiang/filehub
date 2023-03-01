package main

import (
	"filehub/pkg/proto"
	"filehub/pkg/server"
	"log"
	"net"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

var address string
var dataDir string

func main() {
	cmd := NewCommand()
	if err := cmd.Execute(); err != nil {
		log.Fatalf(err.Error())
	}
}

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Run: func(cmd *cobra.Command, args []string) {
			listen, err := net.Listen("tcp", address)
			if err != nil {
				panic(err)
			}

			s := grpc.NewServer()
			log.Println("listening at", address)
			proto.RegisterFileManagerServer(s, server.NewServer(dataDir))
			if err := s.Serve(listen); err != nil {
				panic(err)
			}
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&address, "address", ":9999", "listen address")
	flags.StringVar(&dataDir, "data-dir", "/tmp", "data directory")
	return cmd
}
