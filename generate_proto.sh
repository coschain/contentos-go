#!/bin/bash

CUR_DIR=$(cd "$(dirname "${BASH_SOURCE-$0}")"; pwd)

cd ${CUR_DIR}

# base type proto generate
protoc --go_out=paths=source_relative:. prototype/*.proto

cd ${CUR_DIR}/rpc/pb

# RPC proto generate
protoc -I. -I../.. --go_out=plugins=grpc:. grpc.proto

mockgen -source=grpc.pb.go > ../mock_grpcpb/mock_grpcpb.go
