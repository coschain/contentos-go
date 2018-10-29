package main

import (
	"fmt"
	"github.com/coschain/contentos-go/cmd/cosd/commands"
	"github.com/coschain/contentos-go/node"
	"github.com/spf13/viper"
	"path/filepath"

	"os"
)

func makeConfig() (*node.Node, node.Config) {
	cfg := node.DefaultNodeConfig
	cfg.Name = commands.ClientIdentifier
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	confdir := filepath.Join(cfg.DataDir, cfg.Name)
	viper.AddConfigPath(confdir)
	err := viper.ReadInConfig()
	if err == nil {
		viper.Unmarshal(&cfg)
	} else {
		fmt.Println("Please init cosd first")
		os.Exit(1)
	}
	app, err := node.New(&cfg)
	if err != nil {
		fmt.Println("Fatal: ", err)
		os.Exit(1)
	}
	return app, cfg

}
