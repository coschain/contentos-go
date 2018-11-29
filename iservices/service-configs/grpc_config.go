package service_configs

type GRPCConfig struct {
	RPCListen  string
	HTTPListen string
	HTTPCors   []string
	HTTPLimit  int
}
