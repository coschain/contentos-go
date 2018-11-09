package node

import (
	"github.com/coschain/contentos-go/iservices/service-configs"
	"os"
	"os/user"
	"path/filepath"
)

const (
	DefaultHTTPHost = "localhost"
	DefaultHTTPPort = 8123
)

// DefaultConfig contains reasonable default settings.
var DefaultNodeConfig = Config{
	DataDir:  DefaultDataDir(),
	HTTPHost: DefaultHTTPHost,
	HTTPPort: DefaultHTTPPort,
	Timer: service_configs.TimerConfig{
		Interval: 500,
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
