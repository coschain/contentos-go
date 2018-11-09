package commands

import (
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet/wallet"
)

var AccountCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use: "account",
	}

	getAccountCmd := &cobra.Command{
		Use:   "get",
		Short: "get account info",
		Run:   getAccount,
	}

	cmd.AddCommand(getAccountCmd)

	return cmd
}

func getAccount(cmd *cobra.Command, args []string) {
	_ = args
	o := cmd.Context["wallet"]
	_ = o.(wallet.Wallet)
	fmt.Println("get account")
}
