package iservices

import (
	"github.com/coschain/contentos-go/node"
)

var RPC_SERVER_NAME = "rpc"

type IRPCServer interface {
	Start(node *node.Node) error
	Stop() error
	RunGateway() error
}
