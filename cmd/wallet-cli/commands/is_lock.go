package commands

import (
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
)

var IsLockedCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "locked",
		Short: "whether a account has been locked",
		Run:   isLocked,
	}
	return cmd
}

func isLocked(cmd *cobra.Command, args []string) {
	o := cmd.Context["wallet"]
	w := o.(wallet.Wallet)
	name := args[0]
	l, err := w.IsLocked(name)
	if err != nil {
		fmt.Println(fmt.Sprintf("unknown accout name %s.", name))
		return
	}
	if l {
		fmt.Println(fmt.Sprintf("account %s has been locked", name))
	} else {
		fmt.Println(fmt.Sprintf("account %s has been unlocked", name))
	}
}
