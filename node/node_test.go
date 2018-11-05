package node

import (
	"testing"
)

func testNodeConfig() Config {
	cfg := DefaultNodeConfig
	cfg.Name = "cosd"
	return cfg
}

// Tests that an empty protocol stack can be started, restarted and stopped.
func TestNodeLifeCycle(t *testing.T) {
	cfg := testNodeConfig()
	stack, err := New(&cfg)
	//stack.Register(func(ctx *ServiceContext) (Service, error) {
	//	return timer.New(ctx)
	//})
	//stack.Register(func(ctx *ServiceContext) (Service, error) {
	//	return printer.New(ctx)
	//})
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}
	// Ensure that a node can be successfully started, but only once
	if err := stack.Start(); err != nil {
		t.Fatalf("failed to start node: %v", err)
	}
}
