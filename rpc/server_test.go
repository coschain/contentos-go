package rpc

import (
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/p2p"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGRPCServer(t *testing.T) {
	gs := NewGRPCServer(&node.ServiceContext{})
	err := gs.Start(&p2p.Server{})
	defer gs.Stop()
	assert.Equal(t, nil, err)
}
