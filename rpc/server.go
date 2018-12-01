package rpc

import (
	"github.com/coschain/contentos-go/common/logging"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/iservices/service-configs"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/rpc/pb"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"net"

	"google.golang.org/grpc"
)

const (
	GRPCMaxRecvMsgSize = 4096
	GRPCServerType     = "tcp"
)

type GRPCServer struct {
	rpcServer *grpc.Server
	ctx       *node.ServiceContext
	api       *APIService
	config    *service_configs.GRPCConfig
}

func NewGRPCServer(ctx *node.ServiceContext, config service_configs.GRPCConfig) (*GRPCServer, error) {
	rpc := grpc.NewServer(
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(streamRecoveryLoggingInterceptor)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(unaryRecoveryLoggingInterceptor)),
		grpc.MaxRecvMsgSize(GRPCMaxRecvMsgSize))
	api := &APIService{}
	grpcpb.RegisterApiServiceServer(rpc, api)
	srv := &GRPCServer{rpcServer: rpc, ctx: ctx, api: api, config: &config}
	return srv, nil
}

func (gs *GRPCServer) Start(node *node.Node) error {

	consensus, err := gs.ctx.Service(iservices.ConsensusServerName)
	if err != nil {
		// TODO Mock Test
		//return err
	} else {
		gs.api.consensus = consensus.(iservices.IConsensus)
	}

	db, err := gs.ctx.Service(iservices.DbServerName)
	if err != nil {
		// TODO Mock Test
		//return err
	} else {
		gs.api.db = db.(iservices.IDatabaseService)
	}

	gs.api.mainLoop = node.MainLoop

	err = gs.startGRPC()
	if err != nil {
		return err
	} else {
		logging.CLog().Infof("GPRC Server Start [ %s ]", gs.config.RPCListen)
	}

	err = gs.startGateway()
	if err != nil {
		return err
	} else {
		logging.CLog().Infof("Gateway Server Start [ %s ]", gs.config.HTTPListen)
	}

	return nil
}

func (gs *GRPCServer) startGRPC() error {
	logging.CLog().Infof("RPCListen %v", gs.config.RPCListen)
	listener, err := net.Listen(GRPCServerType, gs.config.RPCListen)
	if err != nil {
		logging.VLog().Errorf("grpc listener addr: [%s] failure", gs.config.RPCListen)
	}

	go func() {
		grpc.NewServer()
		if err := gs.rpcServer.Serve(listener); err != nil {
			logging.VLog().Errorf("rpc server start failure, %v", err)
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

func (gs *GRPCServer) startGateway() error {
	go func() {
		if err := Run(gs.config); err != nil {
			logging.VLog().Error("rpc gateway start failure")
		} else {
			logging.VLog().Info("rpc gateway start failure")
		}
	}()
	return nil
}
