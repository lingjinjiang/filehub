package server

import (
	"context"
	"encoding/json"
	"filehub/pkg/common"
	"filehub/pkg/proto"
	"io"
	"io/fs"
	"log"
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
	destFileInfo := f.files[fileInfo.Name]
	if destFileInfo != nil && destFileInfo.Status == proto.Status_Available {
		return destFileInfo, nil
	}
	return f.ForcePrepare(ctx, fileInfo)
}

func (f *FileManageServerImpl) ForcePrepare(ctx context.Context, fileInfo *proto.FileInfo) (*proto.FileInfo, error) {
	log.Println("Begin Receiving", fileInfo.Name)
	if len(fileInfo.Id) == 0 {
		fileInfo.Id = uuid.New().String()
	}
	destFile, err := os.OpenFile(filepath.Join(f.dataDir, fileInfo.Name+f.tmpSuffix), os.O_RDWR|os.O_CREATE|os.O_TRUNC, fs.FileMode(fileInfo.Perm))
	if err != nil {
		log.Println("Failed to create file", fileInfo.Name, err.Error())
		return nil, err
	}
	defer destFile.Close()
	if err := destFile.Truncate(fileInfo.Size); err != nil {
		log.Println("Failed to truncate file", fileInfo.Name, err.Error())
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
	fileInfo.Status = proto.Status_Available
	f.files[fileInfo.Name] = fileInfo
	saveMetaData(filepath.Join(f.dataDir, metaFile), f.files)
	log.Println("Finish receiving", fileInfo.Name)
	return fileInfo, nil
}

func NewServer(dataDir string) *FileManageServerImpl {
	log.Println("dataDir:", dataDir)
	return &FileManageServerImpl{
		dataDir:   dataDir,
		tmpSuffix: ".tmp",
		files:     loadMetaData(filepath.Join(dataDir, metaFile)),
	}
}

func loadMetaData(filePath string) map[string]*proto.FileInfo {
	files := make(map[string]*proto.FileInfo)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if _, err := os.Create(filePath); err != nil {
			panic(err)
		}
	}
	if data, err := os.ReadFile(filePath); err == nil {
		json.Unmarshal(data, &files)
	}
	if files == nil {
		files = make(map[string]*proto.FileInfo)
	}
	return files
}

func saveMetaData(filePath string, files map[string]*proto.FileInfo) {
	data, _ := json.Marshal(files)
	f, _ := os.OpenFile(filePath, os.O_RDWR, os.ModeAppend)
	f.Write(data)
}
