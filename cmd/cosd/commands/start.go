package commands

import (
	"fmt"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/node"
	log "github.com/inconshreveable/log15"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

var StartCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "start cosd node",
		Run:   startNode,
	}
	cmd.Flags().StringVarP(&cfgName, "name", "n", "", "node name (default is cosd)")
	return cmd
}

func makeNode() (*node.Node, node.Config) {
	cfg := node.DefaultNodeConfig
	if cfgName == "" {
		cfg.Name = ClientIdentifier
	} else {
		cfg.Name = cfgName
	}
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	confdir := filepath.Join(cfg.DataDir, cfg.Name)
	viper.AddConfigPath(confdir)
	err := viper.ReadInConfig()
	if err == nil {
		viper.Unmarshal(&cfg)
	} else {
		fmt.Printf("fatal: not be initialized (do `init` first)\n")
		os.Exit(1)
	}
	if cfg.DataDir != "" {
		dir, err := filepath.Abs(cfg.DataDir)
		if err != nil {
			common.Fatalf("DataDir in cfg cannot be converted to absolute path")
		}
		cfg.DataDir = dir
	}
	app, err := node.New(&cfg)
	if err != nil {
		fmt.Println("Fatal: ", err)
		os.Exit(1)
	}
	return app, cfg

}

// NO OTHER CONFIGS HERE EXCEPT NODE CONFIG
func startNode(cmd *cobra.Command, args []string) {
	// _ is cfg as below process has't used
	_, _ = cmd, args
	app, _ := makeNode()
	if err := app.Start(); err != nil {
		common.Fatalf("start node failed, err: %v\n", err)
	}

	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigc)
		<-sigc
		log.Info("Got interrupt, shutting down...")
		go app.Stop()
	}()
	app.Wait()
}
