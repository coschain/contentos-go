package main

import (
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/cosd/commands"
	"os"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/rpc"
	"github.com/coschain/contentos-go/app"
	"eth/swarm/storage"
)

// cosd is the main entry point into the system if no special subcommand is pointed
// It creates a default node based on the command line arguments and runs it
// in blocking mode, waiting for it to be shut down.
var rootCmd = &cobra.Command{
	Use:   "cosd",
	Short: "Cosd is a fast blockchain designed for content",
}

func addCommands() {
	rootCmd.AddCommand(commands.InitCmd())
	rootCmd.AddCommand(commands.StartCmd())
}

func main() {
	addCommands()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}

	cosNode := initNode()

	// register service ...
	cosNode.Register("db",func(ctx *node.ServiceContext) (node.Service, error) {
		return storage.New(ctx)
	})
	cosNode.Register("rpc", func(ctx *node.ServiceContext) (node.Service, error) {
		return rpc.NewGRPCServer(ctx)
	})
	cosNode.Register("controller", func(ctx *node.ServiceContext) (node.Service, error) {
		return app.NewController(ctx)
	})
}

func initNode() *node.Node {
	cfg := node.DefaultNodeConfig
	cfg.Name = commands.ClientIdentifier
	node, err := node.New(&cfg)
	if err != nil {
		fmt.Println("Fatal: ", err)
		os.Exit(1)
	}
	return node
}
