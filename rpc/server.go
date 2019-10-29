package rpc

import (
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/iservices/service-configs"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/rpc/pb"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net"

	"google.golang.org/grpc"
)

const (
	GRPCMaxRecvMsgSize = 4096*1024
	GRPCServerType     = "tcp"
)

type GRPCServer struct {
	rpcServer *grpc.Server
	ctx       *node.ServiceContext
	api       *APIService
	config    *service_configs.GRPCConfig
	log       *logrus.Logger
}

func NewGRPCServer(ctx *node.ServiceContext, config service_configs.GRPCConfig, lg *logrus.Logger) (*GRPCServer, error) {
	if lg == nil {
		lg = logrus.New()
		lg.SetOutput(ioutil.Discard)
	}

	gi := NewGRPCIntercepter(lg)

	rpc := grpc.NewServer(
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(gi.streamRecoveryLoggingInterceptor)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(gi.unaryRecoveryLoggingInterceptor)),
		grpc.MaxRecvMsgSize(GRPCMaxRecvMsgSize))
	api := &APIService{ctx: ctx}
	grpcpb.RegisterApiServiceServer(rpc, api)
	srv := &GRPCServer{rpcServer: rpc, ctx: ctx, api: api, config: &config}

	srv.log = lg
	srv.api.log = srv.log
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

	pool, err := gs.ctx.Service(iservices.TxPoolServerName)
	if err != nil {
		// TODO Mock Test
		//return err
	} else {
		gs.api.pool = pool.(iservices.ITrxPool)
	}

	p2p, err := gs.ctx.Service(iservices.P2PServerName)
	if err != nil {
		// TODO Mock Test
		//return err
	} else {
		gs.api.p2p = p2p.(iservices.IP2P)
	}

	db, err := gs.ctx.Service(iservices.DbServerName)
	if err != nil {
		// TODO Mock Test
		//return err
	} else {
		gs.api.db = db.(iservices.IDatabaseService)
	}

	gs.api.mainLoop = node.MainLoop
	gs.api.eBus = node.EvBus

	err = gs.startGRPC()
	if err != nil {
		return err
	} else {
		gs.log.Infof("GPRC Server Start [ %s ]", gs.config.RPCListen)
	}


	err = gs.startWebProxy()
	if err != nil {
		return err
	} else {
		gs.log.Infof("WebProxy Server Start [ %s ]", gs.config.HTTPListen)
	}

	return nil
}

func (gs *GRPCServer) startGRPC() error {
	gs.log.Infof("RPCListen %v", gs.config.RPCListen)
	listener, err := net.Listen(GRPCServerType, gs.config.RPCListen)
	if err != nil {
		gs.log.Errorf("grpc listener addr: [%s] failure", gs.config.RPCListen)
	}

	go func() {
		grpc.NewServer()
		if err := gs.rpcServer.Serve(listener); err != nil {
			gs.log.Errorf("rpc server start failure, %v", err)
		} else {
			gs.log.Info("rpc server start failure")
		}
	}()

	return nil
}

func (gs *GRPCServer) Stop() error {
	gs.rpcServer.Stop()
	return nil
}

func (gs *GRPCServer) Reload() error {
	return nil
}

func (gs *GRPCServer) startWebProxy() error {
	go func() {
		if err := RunWebProxy(gs.rpcServer, gs.config); err != nil {
			gs.log.Error("rpc WebProxy start failure")
		} else {
			gs.log.Info("rpc WebProxy start success")
		}
	}()
	return nil
}

func Dial(target string) (*grpc.ClientConn, error) {
	conn, err := grpc.Dial(target, grpc.WithInsecure())
	if err != nil {
		//logging.VLog().Error("rpc.Dial() failed: ", err)
	}
	return conn, err
}
