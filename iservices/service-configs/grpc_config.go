package service_configs

type GRPCConfig struct {
	RPCListen  string   `toml:"0.0.0.0:8888"`
	HTTPListen string   `toml:"0.0.0.0:8080"`
	HTTPCors   []string `toml:",omitempty"`
	HTTPLimit  int      `toml:",omitempty"`
}
