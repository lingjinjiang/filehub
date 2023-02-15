package server

import (
	"context"
	"encoding/json"
	"filehub/pkg/common"
	"filehub/pkg/proto"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

const metaFile string = "meta.json"

type FileManageServerImpl struct {
	proto.UnimplementedFileManagerServer
	dataDir   string
	tmpSuffix string
	files     map[string]*proto.FileInfo
}

func (f *FileManageServerImpl) Prepare(ctx context.Context, fileInfo *proto.FileInfo) (*proto.FileInfo, error) {
	fmt.Println("Begin Receiving", fileInfo.Name)
	if len(fileInfo.Id) == 0 {
		fileInfo.Id = uuid.New().String()
	}
	destFile, err := os.Create(filepath.Join(f.dataDir, fileInfo.Name+f.tmpSuffix))
	if err != nil {
		fmt.Println("Failed to create file", fileInfo.Name, err.Error())
		return nil, err
	}
	defer destFile.Close()
	if err := destFile.Truncate(fileInfo.Size); err != nil {
		fmt.Println("Failed to truncate file", fileInfo.Name, err.Error())
		return nil, err
	}
	fileInfo.BlockSize = common.BLOCK_SIZE
	fileInfo.BlockNum = fileInfo.Size / common.BLOCK_SIZE
	fileInfo.Blocks = make(map[int32]*proto.Block)
	fileInfo.Status = proto.Status_Unavailable
	f.files[fileInfo.Id] = fileInfo
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
	dataDir := "/tmp"

	return &FileManageServerImpl{
		dataDir:   dataDir,
		tmpSuffix: ".tmp",
		files:     loadMetaData(filepath.Join(dataDir, metaFile)),
	}
}

func loadMetaData(filePath string) map[string]*proto.FileInfo {
	files := make(map[string]*proto.FileInfo)
	if data, err := ioutil.ReadFile(filePath); err == nil {
		json.Unmarshal(data, files)
	}
	return files
}
