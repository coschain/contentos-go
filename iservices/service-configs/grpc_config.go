package service_configs

type GRPCConfig struct {
	RPCName    string
	RPCListen  string
	HTTPListen string
	HTTPCors   []string
	HTTPLimit  int
}
