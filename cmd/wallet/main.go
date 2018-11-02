package main

import (
	"fmt"
	"github.com/coschain/contentos-go/cmd/cosd/commands"
	"github.com/spf13/cobra"
	"os"
)

var rootCmd = &cobra.Command{
	Use:   "wallet",
	Short: "wallet is a key-pair storage",
}

func addCommands() {
	rootCmd.AddCommand(commands.InitCmd())
	rootCmd.AddCommand(commands.StartCmd())
}

func main() {
	addCommands()
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
