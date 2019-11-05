package main

import (
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/cosd/commands"
	"os"
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
	rootCmd.AddCommand(commands.DbCmd())
}

func main() {
	addCommands()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}