package rpc

import (
	"fmt"
	"github.com/coschain/contentos-go/rpc/pb"
	"net"

	"google.golang.org/grpc"
)

type RPCServer interface {
	Start() error
	Stop()
}

type GRPCServer struct {
	rpcServer *grpc.Server
}

func NewGRPCServer() *GRPCServer {
	rpc := grpc.NewServer(grpc.MaxRecvMsgSize(4096))
	srv := &GRPCServer{rpcServer:rpc}

	api := &APIService{server:srv}
	grpcpb.RegisterApiServiceServer(rpc, api)

	return srv
}

func (gs *GRPCServer) Start() error {
	err := gs.start("127.0.0.1:8888")
	if err != nil {
		return err
	}
	return nil
}

func (gs *GRPCServer) start(add string) error {
	listener, err := net.Listen("tcp", add)
	if err != nil {
		fmt.Print("listener success")
	}

	go func() {
		if err := gs.rpcServer.Serve(listener); err != nil {
			fmt.Print("rpcServer success")
		}
	}()

	return nil
}

func (gs *GRPCServer) Stop() {
	gs.rpcServer.Stop()
}