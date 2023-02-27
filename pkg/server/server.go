package server

import (
	"context"
	"encoding/json"
	"filehub/pkg/common"
	"filehub/pkg/proto"
	"fmt"
	"io"
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
	if fileInfo.Size%common.BLOCK_SIZE != 0 {
		fileInfo.BlockNum++
	}
	fileInfo.Blocks = make(map[int32]*proto.Block)
	fileInfo.Status = proto.Status_Unavailable
	f.files[fileInfo.Name] = fileInfo
	return fileInfo, nil
}

func (f *FileManageServerImpl) UploadBlock(stream proto.FileManager_UploadBlockServer) error {
	data := make([]byte, 0)
	var filename string
	id := uuid.New().String()
	var sequence uint32
	for {
		block, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		block.Id = id
		sequence = block.Sequence
		filename = block.Filename
		data = append(data, block.Data...)

	}
	destFile, err := os.OpenFile(filepath.Join(f.dataDir, filename+f.tmpSuffix), os.O_RDWR, os.ModeAppend)
	if err != nil {
		return err
	}
	defer destFile.Close()
	if _, err = destFile.WriteAt(data, int64(sequence)*common.BLOCK_SIZE); err != nil {
		return err
	}
	return nil
}

func (f *FileManageServerImpl) Finish(ctx context.Context, fileInfo *proto.FileInfo) (*proto.FileInfo, error) {
	os.Rename(filepath.Join(f.dataDir, fileInfo.Name+f.tmpSuffix), filepath.Join(f.dataDir, fileInfo.Name))
	f.files[fileInfo.Name] = fileInfo
	saveMetaData(f.files)
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
	var files map[string]*proto.FileInfo
	os.Create(filePath)
	if data, err := os.ReadFile(filePath); err == nil {
		json.Unmarshal(data, files)
	}
	if files == nil {
		files = make(map[string]*proto.FileInfo)
	}
	return files
}

func saveMetaData(files map[string]*proto.FileInfo) {
	data, _ := json.Marshal(files)
	f, _ := os.OpenFile(filepath.Join("/tmp", metaFile), os.O_RDWR, os.ModeAppend)
	f.Write(data)
}
