package service_configs

type P2PConfig struct {
	Genesis                   *GenesisConfig
	EnableConsensus           bool
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

type GenesisConfig struct {
	SeedList      []string
}

type P2PRsvConfig struct {
	ReservedPeers []string
	MaskPeers     []string
}