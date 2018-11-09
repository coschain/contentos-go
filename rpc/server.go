package rpc

import (
	"github.com/coschain/contentos-go/common/logging"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/rpc/pb"
	"net"

	"google.golang.org/grpc"
)

type RPCServer interface {
	Start() error
	Stop() error
	RunGateway() error
}

type GRPCServer struct {
	rpcServer *grpc.Server
}

func NewGRPCServer(ctx *node.ServiceContext) (*GRPCServer, error) {
	rpc := grpc.NewServer(grpc.MaxRecvMsgSize(4096))
	srv := &GRPCServer{rpcServer: rpc}

	api := &APIService{server: srv}
	grpcpb.RegisterApiServiceServer(rpc, api)

	return srv, nil
}

func (gs *GRPCServer) Start() error {
	err := gs.start("127.0.0.1:8888")
	if err != nil {
		return err
	}
	return nil
}

func (gs *GRPCServer) start(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		logging.VLog().Errorf("grpc listener addr: [%s] failure", addr)
	}

	go func() {
		if err := gs.rpcServer.Serve(listener); err != nil {
			logging.VLog().Error("rpc server start failure")
		} else {
			logging.VLog().Info("rpc server start failure")
		}
	}()

	return nil
}

func (gs *GRPCServer) Stop() error {
	gs.rpcServer.Stop()
	return nil
}

func (gs *GRPCServer) RunGateway() error {
	go func() {
		if err := Run(); err != nil {
			logging.VLog().Error("rpc gateway start failure")
		} else {
			logging.VLog().Info("rpc gateway start failure")
		}
	}()
	return nil
}
