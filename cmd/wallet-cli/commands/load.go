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
		Args:  cobra.ExactArgs(1),
		Run:   load,
	}
	return cmd
}

var LoadAllCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "loadAll",
		Short: "load all accounts in default path",
		Args:  cobra.ExactArgs(0),
		Run:   loadAll,
	}
	return cmd
}

func load(cmd *cobra.Command, args []string) {
	o := cmd.Context["wallet"]
	w := o.(wallet.Wallet)
	name := args[0]
	err := w.Load(name)
	if err != nil {
		fmt.Println(fmt.Sprintf("error: %v", err))
		return
	}
	fmt.Println(fmt.Sprintf("load account %s success", name))
}

func loadAll(cmd *cobra.Command, args []string) {
	o := cmd.Context["wallet"]
	w := o.(*wallet.BaseWallet)
	err := w.LoadAll()
	if err != nil {
		fmt.Println(fmt.Sprintf("error: %v", err))
	} else {
		fmt.Println(fmt.Sprintf("load all success"))
	}
}
