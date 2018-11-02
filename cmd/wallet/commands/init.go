package commands

import (
	"fmt"
	"github.com/spf13/cobra"
)

var InitCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize configuration files",
		Run:   initConf,
	}
	return cmd
}

func initConf(cmd *cobra.Command, args []string) {
	fmt.Println("hello world")
}
