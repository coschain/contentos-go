package commands

import (
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/config"
	"os"
	"path/filepath"
)

var InitCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize configuration files",
		Run:   initConf,
	}
	cmd.Flags().StringVarP(&cfgName, "name", "n", "", "node name (default is cosd)")
	cmd.Flags().StringVarP(&chainName, "chain", "c", "", "chain name [main/test/dev], default is main")
	return cmd
}

func initConf(cmd *cobra.Command, args []string) {
	_, _ = cmd, args
	var err error
	cfg := config.DefaultNodeConfig
	if cfgName == "" {
		cfg.Name = ClientIdentifier
	} else {
		cfg.Name = cfgName
	}
	if len(chainName) == 0 {
		cfg.ChainId = "main"
	} else {
		cfg.ChainId = chainName
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
