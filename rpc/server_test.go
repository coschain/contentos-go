package rpc

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func setupTestCase(t *testing.T) func(t *testing.T) {
	fmt.Print("setup test case")
	return func(t *testing.T) {
		fmt.Print("teardown test case")
	}
}

func TestGRPCServer(t *testing.T) {
	gs := NewGRPCServer()
	err := gs.Start()
	defer gs.Stop()
	assert.Equal(t, nil, err)
}