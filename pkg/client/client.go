package client

import (
	"context"
	"filehub/pkg/common"
	"filehub/pkg/proto"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const buffer_size = 1024 * 1024

type FClient struct {
	address  string
	threads  int
	conn     *grpc.ClientConn
	fmClient proto.FileManagerClient
}

func NewClient(address string, threads int) *FClient {
	return &FClient{
		address:  address,
		threads:  threads,
		conn:     nil,
		fmClient: nil,
	}
}

func (c *FClient) Connect() error {
	conn, err := grpc.Dial(c.address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	c.conn = conn
	c.fmClient = proto.NewFileManagerClient(conn)
	return nil
}

func (c *FClient) Upload(inputFile string, force bool) (*proto.FileInfo, error) {
	fileInfo, err := c.prepare(inputFile, force)
	if err != nil {
		return nil, err
	}
	if fileInfo.Status == proto.Status_Available {
		fmt.Println(fileInfo.Name, "already exists, skip upload")
		return fileInfo, nil
	}
	c.uploadBlocks(inputFile, fileInfo)
	if err := c.finish(fileInfo); err != nil {
		return nil, err
	}
	return fileInfo, nil
}

func (c *FClient) prepare(inputFile string, force bool) (*proto.FileInfo, error) {
	file, err := os.Open(inputFile)
	if err != nil {
		return nil, err
	}
	stat, _ := file.Stat()
	var fileInfo *proto.FileInfo
	if force {
		fileInfo, err = c.fmClient.ForcePrepare(context.Background(), &proto.FileInfo{
			Name: filepath.Base(inputFile),
			Size: stat.Size(),
			Perm: uint32(stat.Mode()),
		})
	} else {
		fileInfo, err = c.fmClient.Prepare(context.Background(), &proto.FileInfo{
			Name: filepath.Base(inputFile),
			Size: stat.Size(),
			Perm: uint32(stat.Mode()),
		})
	}
	if err != nil {
		return nil, err
	} else {
		return fileInfo, nil
	}
}

func (c *FClient) uploadBlocks(inputFile string, fileInfo *proto.FileInfo) {
	blockCh := make(chan blockInfo, c.threads)
	var uploadSize int64 = 0
	wg := sync.WaitGroup{}
	wg.Add(int(fileInfo.BlockNum))
	for i := 0; i < c.threads; i++ {
		go runStream(c.address, blockCh, &uploadSize, &wg)
	}
	go printProgress(fileInfo.Size, &uploadSize)
	file, err := os.Open(inputFile)
	if err != nil {
		panic(err)
	}
	for i := 0; i < int(fileInfo.BlockNum); i++ {
		dataSize := common.BLOCK_SIZE
		if i == int(fileInfo.BlockNum)-1 && fileInfo.Size-int64(i)*common.BLOCK_SIZE > 0 {
			dataSize = fileInfo.Size - int64(i)*common.BLOCK_SIZE
		}
		blockCh <- blockInfo{
			file:     file,
			filename: fileInfo.Name,
			sequence: i,
			dataSize: dataSize,
		}
	}
	wg.Wait()
	for i := 0; i < c.threads; i++ {
		blockCh <- blockInfo{sequence: -1}
	}
}

func printProgress(size int64, uploadSize *int64) {
	for *uploadSize < size {
		i := *uploadSize * 100 / size
		format := fmt.Sprintf("\r[%s%%-%ds]%%4d%%%%", strings.Repeat("=", int(i/5)), 20-int(i/5))
		last := ">"
		fmt.Printf(format, last, i)
		time.Sleep(200 * time.Millisecond)
	}
	format := fmt.Sprintf("\r[%s%%-%ds]%%4d%%%%", strings.Repeat("=", 20), 0)
	fmt.Printf(format+"\n", "", 100)
}

func runStream(address string, blockCh chan blockInfo, uploadSize *int64, wg *sync.WaitGroup) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	cli := proto.NewFileManagerClient(conn)
	defer func() {
		conn.Close()
	}()
	var blockInfo blockInfo
	for {
		blockInfo = <-blockCh
		if blockInfo.sequence == -1 {
			break
		}
		stream, err := cli.UploadBlock(context.Background())
		if err != nil {
			panic(err)
		}
		offset := int64(blockInfo.sequence) * common.BLOCK_SIZE
		end := offset + blockInfo.dataSize
		for offset < end {
			var data []byte
			if end-offset > buffer_size {
				data = make([]byte, buffer_size)
			} else {
				data = make([]byte, end-offset)

			}
			blockInfo.file.ReadAt(data, offset)
			if err := stream.Send(&proto.Block{
				Sequence: uint32(blockInfo.sequence),
				Filename: blockInfo.filename,
				Data:     data,
			}); err != nil {
				panic(err)
			}
			offset += buffer_size
			*uploadSize += buffer_size
		}
		stream.CloseAndRecv()
		wg.Done()
	}
}

func (c *FClient) finish(fileInfo *proto.FileInfo) error {
	if _, err := c.fmClient.Finish(context.Background(), fileInfo); err != nil {
		return err
	}
	return nil
}

type blockInfo struct {
	sequence int
	dataSize int64
	filename string
	file     *os.File
}
