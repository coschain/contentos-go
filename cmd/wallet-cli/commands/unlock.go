package commands

import (
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet-cli/commands/utils"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
)

var UnlockCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "unlock",
		Short:   "unlock a account",
		Example: "unlock [name]",
		Args:    cobra.ExactArgs(1),
		Run:     unlock,
	}
	return cmd
}

func unlock(cmd *cobra.Command, args []string) {
	o := cmd.Context["wallet"]
	w := o.(wallet.Wallet)
	name := args[0]
	r := cmd.Context["preader"]
	preader := r.(utils.PasswordReader)
	passphrase, err := utils.GetPassphrase(preader)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = w.Unlock(name, passphrase)
	if err != nil {
		fmt.Println(fmt.Sprintf("error: %v", err))
		return
	}
	fmt.Println(fmt.Sprintf("unlock account %s success", name))
}
