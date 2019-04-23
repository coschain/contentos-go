package service_configs

type DatabaseConfig struct {
	Driver   string `toml:",omitempty"`
	User     string `toml:"-"`
	Password string	`toml:"-"`
	Db       string `toml:",omitempty"`
}
