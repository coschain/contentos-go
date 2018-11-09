package iservices

import (
	"github.com/coschain/contentos-go/node"
)


var RPC_SERVER_NAME = "rpc"

type RPCServer interface {
	Start(config *node.Config) error
	Stop() error
	RunGateway() error
}