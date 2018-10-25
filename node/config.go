package node

import (
	"fmt"
	"github.com/coschain/contentos-go/p2p"
	log "github.com/inconshreveable/log15"
	"path/filepath"
	"runtime"
)

const (
	datadirDatabase = "nodes"
)

type Config struct {
	// Name refers the name of node's instance
	Name string `toml:"-"`

	// Version should be set to the version number of the program.
	Version string `toml:"-"`

	// configuration of p2p networking
	P2P p2p.Config

	// DataDir is the root folder that store data and configs
	DataDir string

	// HTTPHost is the host interface on which to start the HTTP RPC server.
	HTTPHost string `toml:",omitempty"`

	// HTTPPort is the TCP port number on which to start the HTTP RPC server.
	HTTPPort int `toml:",omitempty"`

	// Logger is a custom logger
	Logger log.Logger `toml:",omitempty"`
}

// DB returns the path to the discovery database.
func (c *Config) NodeDB() string {
	if c.DataDir == "" {
		return ""
	}
	return c.ResolvePath(datadirDatabase)
}

// HTTPEndpoint resolves an HTTP endpoint based on the configured host interface
// and port parameters.
func (c *Config) HTTPEndpoint() string {
	if c.HTTPHost == "" {
		return ""
	}
	return fmt.Sprintf("%s:%d", c.HTTPHost, c.HTTPPort)
}

//// DefaultHTTPEndpoint returns the HTTP endpoint used by default.
//func DefaultHTTPEndpoint() string {
//	config := &Config{HTTPHost: DefaultHTTPHost, HTTPPort: DefaultHTTPPort}
//	return config.HTTPEndpoint()
//}

func (c *Config) name() string {
	if c.Name == "" {
		panic("empty node name, set Config.Name")
	}
	return c.Name
}

// GetName returns the node's complete name
func (c *Config) NodeName() string {
	name := c.name()
	if c.Version != "" {
		name += "/v" + c.Version
	}
	name += "/" + runtime.GOOS + "-" + runtime.GOARCH
	name += "/" + runtime.Version()
	return name
}

// ResolvePath resolves path in the instance directory.
func (c *Config) ResolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	if c.DataDir == "" {
		return ""
	}
	return filepath.Join(c.instanceDir(), path)
}

func (c *Config) instanceDir() string {
	if c.DataDir == "" {
		return ""
	}
	return filepath.Join(c.DataDir, c.Name)
}
