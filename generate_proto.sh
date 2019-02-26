#!/bin/bash

CUR_DIR=$(cd "$(dirname "${BASH_SOURCE-$0}")"; pwd)

cd ${CUR_DIR}

# base type proto generate
protoc --go_out=paths=source_relative:. prototype/*.proto

cd ${CUR_DIR}/rpc/pb

# RPC proto generate
protoc -I/usr/local/include -I. -I$GOPATH/src -I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis -I$GOPATH/src/github.com/coschain/contentos-go --go_out=plugins=grpc:. grpc.proto
protoc -I/usr/local/include -I. -I$GOPATH/src -I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis -I$GOPATH/src/github.com/coschain/contentos-go --grpc-gateway_out=logtostderr=true:. grpc.proto

mockgen -source=grpc.pb.go > ../mock_grpcpb/mock_grpcpb.go
