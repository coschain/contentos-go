package node

import (
	"github.com/coschain/contentos-go/config"
	"testing"
)

func testNodeConfig() Config {
	cfg := config.DefaultNodeConfig
	cfg.Name = "cosd"
	return cfg
}

// Tests that an empty protocol stack can be started, restarted and stopped.
func TestNodeLifeCycle(t *testing.T) {
	cfg := testNodeConfig()
	stack, err := New(&cfg)
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}
	// Ensure that a node can be successfully started, but only once
	if err := stack.Start(); err != nil {
		t.Fatalf("failed to start node: %v", err)
	}
	if err := stack.Restart(); err != nil {
		t.Fatalf("failed to restart node: %v", err)
	}
	if err := stack.Stop(); err != nil {
		t.Fatalf("failed to stop node: %v", err)
	}

}
