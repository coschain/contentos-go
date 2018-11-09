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
		Run:   list,
	}
	return cmd
}

func list(cmd *cobra.Command, args []string) {
	_ = args
	o := cmd.Context["wallet"]
	w := o.(wallet.Wallet)
	lines := w.List()
	for _, line := range lines {
		fmt.Println(line)
	}
}
