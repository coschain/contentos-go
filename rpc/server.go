package rpc

import (
	"github.com/coschain/contentos-go/common/logging"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/iservices/service-configs"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/rpc/pb"
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
	rpc := grpc.NewServer(grpc.MaxRecvMsgSize(GRPCMaxRecvMsgSize))

	api := &APIService{}

	grpcpb.RegisterApiServiceServer(rpc, api)

	srv := &GRPCServer{rpcServer: rpc, ctx: ctx, api: api, config: &config}

	return srv, nil
}

func (gs *GRPCServer) Start(node *node.Node) error {

	ctrl, err := gs.ctx.Service(iservices.CTRL_SERVER_NAME)
	if err != nil {
		// TODO Mock Test
		//return err
	} else {
		gs.api.ctrl = ctrl.(iservices.IController)
	}

	db, err := gs.ctx.Service(iservices.DB_SERVER_NAME)
	if err != nil {
		// TODO Mock Test
		//return err
	} else {
		gs.api.db = db.(iservices.IDatabaseService)
	}

	gs.api.mainLoop = node.MainLoop

	err = gs.start(gs.config.RPCListeners)
	if err != nil {
		return err
	} else {
		logging.CLog().Info("GPRC server start ...")
	}
	return nil
}

func (gs *GRPCServer) start(addr string) error {
	listener, err := net.Listen(GRPCServerType, addr)
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
		if err := Run(gs.config); err != nil {
			logging.VLog().Error("rpc gateway start failure")
		} else {
			logging.VLog().Info("rpc gateway start failure")
		}
	}()
	return nil
}
