package commands

import (
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
)

var LoadCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "load",
		Short: "load a created account",
		Run:   load,
	}
	return cmd
}

func load(cmd *cobra.Command, args []string) {
	o := cmd.Context["wallet"]
	w := o.(wallet.Wallet)
	name := args[0]
	err := w.Load(name)
	if err != nil {
		fmt.Println(fmt.Sprintf("error: %v\n", err))
		return
	}
	fmt.Println(fmt.Sprintf("load account %s success", name))
}
