package main

import (
	"filehub/pkg/client"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var (
	server  string
	threads int
)

func main() {
	cmd := NewCommand()
	if err := cmd.Execute(); err != nil {
		log.Fatalf(err.Error())
	}

}

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Run: func(cmd *cobra.Command, args []string) {
			defer func() {
				if err := recover(); err != nil {
					fmt.Println(err)
				}
			}()
			fmt.Println(args)
			if len(args) < 1 {
				cmd.Help()
				return
			}
			inputFile := args[0]
			cli := client.NewClient(server, threads)
			if err := cli.Connect(); err != nil {
				panic(err)
			}
			fileInfo, err := cli.Upload(inputFile)
			if err != nil {
				panic(err)
			}
			fmt.Println("Uploaded file", fileInfo.Name, fileInfo.Id)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&server, "server", "127.0.0.1:9999", "server's address")
	flags.IntVarP(&threads, "threads", "t", 3, "threads to upload file")
	return cmd
}
