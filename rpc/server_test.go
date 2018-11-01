package rpc

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGRPCServer(t *testing.T) {
	gs := NewGRPCServer()
	err := gs.Start()
	defer gs.Stop()
	assert.Equal(t, nil, err)
}