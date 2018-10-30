package commands

import (
	"fmt"
	"github.com/coschain/contentos-go/config"
	"github.com/coschain/contentos-go/node"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

var cfgName string

var InitCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize configuration files",
		Run:   initConf,
	}
	cmd.Flags().StringVarP(&cfgName, "name", "n", "", "node name (default is cosd)")
	return cmd
}

func initConf(cmd *cobra.Command, args []string) {
	var err error
	cfg := node.DefaultNodeConfig
	if cfgName == "" {
		cfg.Name = ClientIdentifier
	} else {
		cfg.Name = cfgName
	}
	confdir := filepath.Join(cfg.DataDir, cfg.Name)
	if _, err = os.Stat(confdir); os.IsNotExist(err) {
		if err = os.MkdirAll(confdir, 0700); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
	err = config.WriteNodeConfigFile(confdir, "config.toml", cfg, 0600)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
