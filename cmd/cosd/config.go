package main

import (
	"fmt"
	"github.com/coschain/contentos-go/node"
	"github.com/spf13/viper"

	"os"
)

const (
	clientIdentifier = "cosd"
)

func makeConfig() (*node.Node, node.Config) {
	cfg := node.DefaultNodeConfig
	cfg.Name = clientIdentifier
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(cfg.DataDir)
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
