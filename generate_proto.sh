#!/bin/bash

CUR_DIR=$(cd "$(dirname "${BASH_SOURCE-$0}")"; pwd)

cd ${CUR_DIR}

# install or upgrade dependencies
if [ "$1" == "update" ] ; then
  echo 'update brew and protobuf'

  if brew ls --versions protobuf > /dev/null; then
      brew upgrade protobuf
  else
      brew install protobuf
  fi
  go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
  go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger
  go get github.com/golang/protobuf/protoc-gen-go@v1.3.2
  go get -u github.com/golang/mock/gomock
  go get -u github.com/golang/mock/mockgen
fi

# base type proto generate
protoc --go_out=paths=source_relative:. prototype/*.proto

cd ${CUR_DIR}/rpc/pb

# RPC proto generate
protoc -I. -I../.. --go_out=plugins=grpc:. grpc.proto

mockgen -source=grpc.pb.go > ../mock_grpcpb/mock_grpcpb.go
