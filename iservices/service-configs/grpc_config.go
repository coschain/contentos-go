package service_configs

type GRPCConfig struct {
	RPCListeners string   `toml:"0.0.0.0:8888"`
	HTTPLiseners string   `toml:"0.0.0.0:8080"`
	HTTPCors     []string `toml:",omitempty"`
	HTTPLimit    int      `toml:",omitempty"`
}
