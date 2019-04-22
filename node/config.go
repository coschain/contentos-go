package node

import (
	"github.com/coschain/contentos-go/iservices/service-configs"
	"path/filepath"
	"runtime"
)

const (
	datadirDatabase = "nodes"
)

type Config struct {
	// Name refers the name of node's instance
	Name string

	// Version should be set to the version number of the program.
	Version string `toml:"-"`

	// DataDir is the root folder that store data and service-configs
	DataDir string `toml:",omitempty"`

	// HTTPHost is the host interface on which to start the HTTP RPC server.
	HTTPHost string `toml:",omitempty"`

	// HTTPPort is the TCP port number on which to start the HTTP RPC server.
	HTTPPort int `toml:",omitempty"`

	LogLevel string `toml:",omitempty"`

	ResourceCheck bool `toml:",omitempty"`

	// Logger is a custom logger
	//Logger log.Logger `toml:"-"`

	GRPC service_configs.GRPCConfig

	Consensus service_configs.ConsensusConfig

	P2P   service_configs.P2PConfig
}

// DB returns the path to the discovery database.
func (c *Config) NodeDB() string {
	if c.DataDir == "" {
		return ""
	}
	return c.ResolvePath(datadirDatabase)
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
