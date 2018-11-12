package config

import (
	"github.com/coschain/contentos-go/iservices/service-configs"
	"github.com/coschain/contentos-go/node"
	"os"
	"os/user"
	"path/filepath"
)

const (
	DefaultRPCEndPoint  = "localhost:8888"
	DefaultHTTPEndPoint = "localhost:8080"
)

// DefaultConfig contains reasonable default settings.
var DefaultNodeConfig = node.Config{
	DataDir: DefaultDataDir(),
	Timer: service_configs.TimerConfig{
		Interval: 500,
	},
	GRPC: service_configs.GRPCConfig{
		RPCListeners: DefaultRPCEndPoint,
		HTTPLiseners: DefaultHTTPEndPoint,
	},
}

func DefaultDataDir() string {
	home := homeDir()
	if home != "" {
		return filepath.Join(home, ".coschain")
	}
	return ""
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}
