package service_configs

type DatabaseConfig struct {
	Driver   string `toml:",omitempty"`
	User     string `toml:",omitempty"`
	Password string	`toml:",omitempty"`
	Db       string `toml:",omitempty"`
}
