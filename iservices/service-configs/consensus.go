package service_configs

type ConsensusConfig struct {
	Type      string `toml:",DPoS"`
	BootStrap bool   `toml:",false"`

	//LocalBpName       string `toml:",omitempty"`
	//LocalBpPrivateKey string `toml:",omitempty"`
	LocalBpName       string
	LocalBpPrivateKey string
}
