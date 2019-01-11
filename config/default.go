package config

import (
	"fmt"
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

const (
	DEFAULT_NODE_PORT                       = uint(20338)
	DEFAULT_CONSENSUS_PORT                  = uint(20339)
	DEFAULT_MAX_CONN_IN_BOUND               = uint(1024)
	DEFAULT_MAX_CONN_OUT_BOUND              = uint(1024)
	DEFAULT_MAX_CONN_IN_BOUND_FOR_SINGLE_IP = uint(16)
)

const (
	NETWORK_ID_MAIN_NET    = 1
	NETWORK_ID_TESTNET_NET = 2
	NETWORK_NAME_MAIN_NET  = "contentos"
	NETWORK_NAME_TEST_NET  = "contentos_test"
)

var TestNetConfig = &service_configs.GenesisConfig{
	SeedList: []string{
		fmt.Sprintf("127.0.0.1:%d", DEFAULT_NODE_PORT)},
}

var MainNetConfig = &service_configs.GenesisConfig{
	SeedList: []string{
		fmt.Sprintf("127.0.0.1:%d", DEFAULT_NODE_PORT)},
}

var NETWORK_MAGIC = map[uint32]uint32{
	NETWORK_ID_MAIN_NET:    0x8c77ab66, //Network main
	NETWORK_ID_TESTNET_NET: 0x2d8829ff, //Network testnet
}

var NETWORK_NAME = map[uint32]string{
	NETWORK_ID_MAIN_NET:    NETWORK_NAME_MAIN_NET,
	NETWORK_ID_TESTNET_NET: NETWORK_NAME_TEST_NET,
}

func GetNetworkMagic(id uint32) uint32 {
	nid, ok := NETWORK_MAGIC[id]
	if ok {
		return nid
	}
	return id
}

func GetNetworkName(id uint32) string {
	name, ok := NETWORK_NAME[id]
	if ok {
		return name
	}
	return fmt.Sprintf("%d", id)
}

// DefaultConfig contains reasonable default settings.
var DefaultNodeConfig = node.Config{
	DataDir: DefaultDataDir(),
	LogLevel:         DefaultLogLevel,
	//P2PPort:          20200,
	//P2PPortConsensus: 20201,
	//P2PSeeds:         []string{},
	P2P: service_configs.P2PConfig{
		Genesis:                   MainNetConfig,
		EnableConsensus:           true,
		ReservedCfg:               &service_configs.P2PRsvConfig{},
		ReservedPeersOnly:         false,
		NetworkId:                 NETWORK_ID_MAIN_NET,
		NetworkName:               GetNetworkName(NETWORK_ID_MAIN_NET),
		NetworkMagic:              GetNetworkMagic(NETWORK_ID_MAIN_NET),
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
