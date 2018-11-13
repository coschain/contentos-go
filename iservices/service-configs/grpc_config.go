package service_configs

type GRPCConfig struct {
	RPCListen  string   `toml:"127.0.0.1:8888"`
	HTTPListen string   `toml:"127.0.0.1:8080"`
	HTTPCors   []string `toml:",omitempty"`
	HTTPLimit  int      `toml:",omitempty"`
}
