package main

import (
	"fmt"
	"github.com/coschain/contentos-go/cmd/cosd/commands"
	"github.com/coschain/contentos-go/node"
	"github.com/ethereum/go-ethereum/cmd/utils"
	log "github.com/inconshreveable/log15"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

func makeNode() (*node.Node, node.Config) {
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
		fmt.Printf("fatal: not be initialized (do `init` first)\n")
		os.Exit(1)
	}
	if cfg.DataDir != "" {
		dir, err := filepath.Abs(cfg.DataDir)
		if err != nil {
			utils.Fatalf("DataDir in cfg cannot be converted to absolute path")
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
func cmdStartNode(cmd *cobra.Command, args []string) {
	// _ is cfg as below process has't used
	app, _ := makeNode()
	if err := app.Start(); err != nil {
		utils.Fatalf("start node failed, err: %v\n", err)
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

// cosd is the main entry point into the system if no special subcommand is pointed
// It creates a default node based on the command line arguments and runs it
// in blocking mode, waiting for it to be shut down.
var rootCmd = &cobra.Command{
	Use:   "cosd",
	Short: "Cosd is a fast blockchain designed for content",
	Run:   cmdStartNode,
}

func addCommands() {
	rootCmd.AddCommand(commands.InitCmd)
}

func main() {
	addCommands()
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
