package commands

import (
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet/wallet"
)

var CreateCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "create a new account",
		Run:   create,
	}
	return cmd
}

func create(cmd *cobra.Command, args []string) {
	o := cmd.Context["wallet"]
	w := o.(wallet.Wallet)
	name := args[0]
	passphrase := args[1]
	err := w.Create(name, passphrase)
	if err != nil {
		fmt.Println(fmt.Sprintf("error: %v\n", err))
		return
	}
	fmt.Println(fmt.Sprintf("create account %s success", name))
}
