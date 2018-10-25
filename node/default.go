package node

import (
	"github.com/coschain/contentos-go/p2p"
	"github.com/coschain/contentos-go/p2p/nat"
	"os"
	"os/user"
	"path/filepath"
)

const (
	DefaultHTTPHost = "localhost"
	DefaultHTTPPort = 8123
)

var DefaultNodeConfig = Config{
	DataDir:  DefaultDataDir(),
	HTTPHost: DefaultHTTPHost,
	HTTPPort: DefaultHTTPPort,
	P2P: p2p.Config{
		ListenAddr: ":30303",
		MaxPeers:   25,
		NAT:        nat.Any(),
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
