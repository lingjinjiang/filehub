BASE_DIR = $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
BIN_DIR = ${BASE_DIR}/bin
PKG_DIR = ${BASE_DIR}/pkg
CMD_DIR = ${PKG_DIR}/cmd
SERVER_CMD_DIR = ${CMD_DIR}/server
SERVER_CMD = ${BIN_DIR}/filehub-server
CLIENT_CMD_DIR = ${CMD_DIR}/client
CLIENT_CMD = ${BIN_DIR}/filehub-client
BUILD_TIME = ${shell date +"%Y-%m-%d %Z %T"}

.PHONY: proto build-client build-server build

proto:
	@echo build protobuf
	@rm -rf pkg/proto
	@protoc --go_out=. \
	--proto_path=proto \
	--go_opt=module=filehub \
	--go-grpc_out=. \
	--go-grpc_opt=module=filehub \
	proto/*

build-client: proto
	@echo build client
	@mkdir -p ${BIN_DIR}
	@rm -f ${CLIENT_CMD}
	go build -o ${CLIENT_CMD} ${CLIENT_CMD_DIR}

build-server: proto
	@echo build server
	@mkdir -p ${BIN_DIR}
	@rm -f ${SERVER_CMD}
	go build -o ${SERVER_CMD} ${SERVER_CMD_DIR}

build: build-client build-server