package main

import (
	"os"

	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/databackup/commands"
)

var rootCmd = &cobra.Command{
	Use:   "databackup",
	Short: "databackup sends node data to backup server",
}

func addCommands() {
	rootCmd.AddCommand(commands.AgentCmd())
	rootCmd.AddCommand(commands.ServerCmd())
	rootCmd.AddCommand(commands.ClientCmd())
}

func main() {
	addCommands()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
