package commands

import (
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet/wallet"
)

var UnlockCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unlock",
		Short: "unlock a account",
		Run:   unlockAccount,
	}
	return cmd
}

func unlockAccount(cmd *cobra.Command, args []string) {
	o := cmd.Context["wallet"]
	w := o.(*wallet.BaseWallet)
	name := args[0]
	passphrase := args[1]
	err := w.Unlock(name, passphrase)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}
	fmt.Println(fmt.Sprintf("unlock account %s success", name))
}
