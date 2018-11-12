package iservices

import (
	"github.com/coschain/contentos-go/node"
)

var RPC_SERVER_NAME = "rpc"

type RPCServer interface {
	Start(node *node.Node) error
	Stop() error
	RunGateway() error
}
