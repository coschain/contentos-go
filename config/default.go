package config

import (
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/iservices/service-configs"
	"github.com/coschain/contentos-go/node"
	"github.com/pelletier/go-toml"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
)

const (
	DefaultRPCEndPoint  = "127.0.0.1:8888"
	DefaultHTTPEndPoint = "127.0.0.1:8080"
	DefaultLogLevel     = "debug"
)

// DefaultConfig contains reasonable default settings.
var DefaultNodeConfig = node.Config{
	DataDir: DefaultDataDir(),
	Timer: service_configs.TimerConfig{
		Interval: 500,
	},
	LogLevel:         DefaultLogLevel,
	P2PPort:          20200,
	P2PPortConsensus: 20201,
	P2PSeeds:         []string{},

	GRPC: service_configs.GRPCConfig{
		RPCListen:  DefaultRPCEndPoint,
		HTTPListen: DefaultHTTPEndPoint,
		HTTPCors:   []string{"*"},
		HTTPLimit:  100,
	},
	Consensus: service_configs.ConsensusConfig{
		Type:              "DPoS",
		BootStrap:         true,
		LocalBpName:       constants.INIT_MINER_NAME,
		LocalBpPrivateKey: constants.INITMINER_PRIKEY,
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

func WriteNodeConfigFile(configDirPath string, configName string, config node.Config, mode os.FileMode) error {

	configPath := filepath.Join(configDirPath, configName)
	buffer, err := toml.Marshal(config)

	if err != nil {
		return err
	}

	return ioutil.WriteFile(configPath, buffer, mode)
}
