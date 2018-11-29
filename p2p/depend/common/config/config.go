package config

import (
	"fmt"

	"github.com/coschain/contentos-go/p2p/depend/common/log"
)

var Version = "" //Set value when build project

const (
	DEFAULT_LOG_LEVEL                       = log.InfoLog
	DEFAULT_NODE_PORT                       = uint(20338)
	DEFAULT_CONSENSUS_PORT                  = uint(20339)
	DEFAULT_MAX_CONN_IN_BOUND               = uint(1024)
	DEFAULT_MAX_CONN_OUT_BOUND              = uint(1024)
	DEFAULT_MAX_CONN_IN_BOUND_FOR_SINGLE_IP = uint(16)
)

const (
	NETWORK_ID_MAIN_NET      = 1
	NETWORK_ID_TESTNET_NET   = 2
	NETWORK_NAME_MAIN_NET    = "contentos"
	NETWORK_NAME_TEST_NET    = "contentos_test"
)

var NETWORK_MAGIC = map[uint32]uint32{
	NETWORK_ID_MAIN_NET:    0x8c77ab66,     //Network main
	NETWORK_ID_TESTNET_NET: 0x2d8829ff,     //Network testnet
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

var TestNetConfig = &GenesisConfig{
	SeedList: []string{
		"10.66.108.138:20338"},
}

var MainNetConfig = &GenesisConfig{
	SeedList: []string{
		"127.0.0.1:20338"},
}

var DefConfig = NewContentosConfig()

type GenesisConfig struct {
	SeedList      []string
	ConsensusType string
}

func NewGenesisConfig() *GenesisConfig {
	return &GenesisConfig{
		SeedList: make([]string, 0),
	}
}

type CommonConfig struct {
	LogLevel       uint
}

type ConsensusConfig struct {
	EnableConsensus bool
}

type P2PRsvConfig struct {
	ReservedPeers []string `json:"reserved"`
	MaskPeers     []string `json:"mask"`
}

type P2PNodeConfig struct {
	ReservedPeersOnly         bool
	ReservedCfg               *P2PRsvConfig
	NetworkMagic              uint32
	NetworkId                 uint32
	NetworkName               string
	NodePort                  uint
	NodeConsensusPort         uint
	DualPortSupport           bool
	IsTLS                     bool
	CertPath                  string
	KeyPath                   string
	CAPath                    string
	MaxConnInBound            uint
	MaxConnOutBound           uint
	MaxConnInBoundForSingleIP uint
}

type ContentosConfig struct {
	Genesis   *GenesisConfig
	Common    *CommonConfig
	Consensus *ConsensusConfig
	P2PNode   *P2PNodeConfig
}

func NewContentosConfig() *ContentosConfig {
	return &ContentosConfig{
		Genesis: MainNetConfig,
		Common: &CommonConfig{
			LogLevel:       DEFAULT_LOG_LEVEL,
		},
		Consensus: &ConsensusConfig{
			EnableConsensus: true,
		},
		P2PNode: &P2PNodeConfig{
			ReservedCfg:               &P2PRsvConfig{},
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
	}
}
