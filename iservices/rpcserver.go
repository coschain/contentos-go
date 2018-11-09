package iservices

import "github.com/coschain/contentos-go/p2p"


var RPC_SERVER_NAME = "rpc"

type RPCServer interface {
	Start(server *p2p.Server) error
	Stop() error
	RunGateway() error
}