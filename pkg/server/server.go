package server

import (
	"context"
	"filehub/pkg/common"
	"filehub/pkg/proto"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

type FileManageServerImpl struct {
	proto.UnimplementedFileManagerServer
	dataDir   string
	tmpSuffix string
}

func (f *FileManageServerImpl) Prepare(ctx context.Context, fileInfo *proto.FileInfo) (*proto.FileInfo, error) {
	fmt.Println("Begin Receiving", fileInfo.Name)
	if len(fileInfo.Id) == 0 {
		fileInfo.Id = uuid.New().String()
	}
	destFile, err := os.Create(filepath.Join(f.dataDir, fileInfo.Name+f.tmpSuffix))
	if err != nil {
		fmt.Println("Failed to create file", fileInfo.Name, err.Error())
		return fileInfo, err
	}
	defer destFile.Close()
	if err := destFile.Truncate(fileInfo.Size); err != nil {
		fmt.Println("Failed to truncate file", fileInfo.Name, err.Error())
		return fileInfo, err
	}
	return fileInfo, nil
}

func (f *FileManageServerImpl) UploadBlock(stream proto.FileManager_UploadBlockServer) error {
	for {
		block, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		destFile, err := os.OpenFile(filepath.Join(f.dataDir, block.Filename+f.tmpSuffix), os.O_RDWR, os.ModeAppend)
		if err != nil {
			return err
		}
		defer destFile.Close()
		block.Id = uuid.New().String()
		if _, err = destFile.WriteAt(block.Data, int64(block.Sequence)*common.BLOCK_SIZE); err != nil {
			return err
		}
	}
	return nil
}

func (f *FileManageServerImpl) Finish(ctx context.Context, fileInfo *proto.FileInfo) (*proto.FileInfo, error) {
	os.Rename(filepath.Join(f.dataDir, fileInfo.Name+f.tmpSuffix), filepath.Join(f.dataDir, fileInfo.Name))
	fmt.Println("Finish receiving", fileInfo.Name)
	return fileInfo, nil
}

func NewServer() *FileManageServerImpl {
	return &FileManageServerImpl{
		dataDir:   "/tmp",
		tmpSuffix: ".tmp",
	}
}
