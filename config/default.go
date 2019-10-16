package config

import (
	"fmt"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/iservices/service-configs"
	"github.com/coschain/contentos-go/mylog"
	"github.com/coschain/contentos-go/node"
	"github.com/pelletier/go-toml"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
)

const (
	DefaultRPCEndPoint  = "0.0.0.0:8888"
	DefaultHTTPEndPoint = "0.0.0.0:8080"
)

const (
	DEFAULT_NODE_PORT                       = uint(20338)
	DEFAULT_CONSENSUS_PORT                  = uint(20339)
	DEFAULT_MAX_CONN_IN_BOUND               = uint(23)
	DEFAULT_MAX_CONN_OUT_BOUND              = uint(23)
	DEFAULT_MAX_CONN_IN_BOUND_FOR_SINGLE_IP = uint(23)
)

var TestNetConfig = &service_configs.GenesisConfig{
	SeedList: []string{
		fmt.Sprintf("127.0.0.1:%d", DEFAULT_NODE_PORT)},
}

var DevNetConfig = &service_configs.GenesisConfig{
	SeedList: []string{
		fmt.Sprintf("127.0.0.1:%d", DEFAULT_NODE_PORT)},
}

var MainNetConfig = &service_configs.GenesisConfig{
	SeedList: []string{
		fmt.Sprintf("127.0.0.1:%d", DEFAULT_NODE_PORT)},
}

// DefaultConfig contains reasonable default settings.
var DefaultNodeConfig = node.Config{
	ChainId: common.ChainNameMainNet,
	DataDir: DefaultDataDir(),
	LogLevel:         mylog.DebugLevel,
	P2P: service_configs.P2PConfig{
		Genesis:                   MainNetConfig,
		EnableConsensus:           true,
		ReservedCfg:               &service_configs.P2PRsvConfig{},
		ReservedPeersOnly:         false,
		NodePort:                  DEFAULT_NODE_PORT,
		NodeConsensusPort:         DEFAULT_CONSENSUS_PORT,
		DualPortSupport:           true,
		IsTLS:                     false,
		CertPath:                  "",
		KeyPath:                   "",
		CAPath:                    "",
		MaxConnInBound:            DEFAULT_MAX_CONN_IN_BOUND,
		MaxConnOutBound:           DEFAULT_MAX_CONN_OUT_BOUND,
		MaxConnInBoundForSingleIP: DEFAULT_MAX_CONN_IN_BOUND_FOR_SINGLE_IP,
	},

	GRPC: service_configs.GRPCConfig{
		RPCName:    "rpc",
		RPCListen:  DefaultRPCEndPoint,
		HTTPListen: DefaultHTTPEndPoint,
		HTTPCors:   []string{"*"},
		HTTPLimit:  100,
	},
	Consensus: service_configs.ConsensusConfig{
		Type:              "SABFT",
		BootStrap:         true,
		LocalBpName:       constants.COSInitMiner,
		LocalBpPrivateKey: constants.InitminerPrivKey,
	},
	HealthCheck: service_configs.HCheck{
		Port: "9090",
	},
	// only for local or testing environment
	Database: &service_configs.DatabaseConfig{
		Driver: "mysql",
		User: "contentos",
		Password: "123456",
		Db: "contentosdb",
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

func MakeName(name, version string) string {
	return fmt.Sprintf("%s/%s/%s/%s", name, version, runtime.GOOS, runtime.Version())
}
