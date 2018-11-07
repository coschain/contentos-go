package commands

import (
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet/wallet"
)

var ListCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list all accounts",
		Run:   listAccounts,
	}
	return cmd
}

func listAccounts(cmd *cobra.Command, args []string) {
	o := cmd.Context["wallet"]
	w := o.(*wallet.BaseWallet)
	lines := w.List()
	for _, line := range lines {
		fmt.Println(line)
	}
}
