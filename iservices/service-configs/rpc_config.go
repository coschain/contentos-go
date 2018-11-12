package service_configs

type RPCConfig struct {
	RPCListeners string   `toml:"0.0.0.0:8888"`
	HTTPLiseners string   `toml:"0.0.0.0:8080"`
	HTTPCors     []string `toml:",omitempty"`
}
