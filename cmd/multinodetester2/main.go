package main

import (
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/multinodetester2/commands"
	"os"
)

// cosd is the main entry point into the system if no special subcommand is pointed
// It creates a default node based on the command line arguments and runs it
// in blocking mode, waiting for it to be shut down.
var rootCmd = &cobra.Command{
	Use:   "multinodetester2",
	Short: "multinodetester2 can manager multi-cosd process and config for DPOS-p2p debug",
}

func addCommands() {
	rootCmd.AddCommand(commands.InitCmd())
	rootCmd.AddCommand(commands.StartCmd())
	rootCmd.AddCommand(commands.StopCmd())
	rootCmd.AddCommand(commands.ClearCmd())
}

func main() {
	addCommands()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
