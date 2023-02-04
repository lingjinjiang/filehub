package main

import (
	"filehub/pkg/client"
	"fmt"
	"os"
	"strconv"
)

func main() {
	args := os.Args
	if len(args) < 3 {
		return
	}
	inputFile := args[1]
	threadNum, err := strconv.Atoi(args[2])
	if err != nil {
		panic(err)
	}
	cli := client.NewClient("127.0.0.1:9999", threadNum)
	if err := cli.Connect(); err != nil {
		panic(err)
	}
	fileInfo, err := cli.Upload(inputFile)
	if err != nil {
		panic(err)
	}
	fmt.Println("Upload file", fileInfo.Name, fileInfo.Id)
}
