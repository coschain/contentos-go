package commands

import (
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet/wallet"
)

var CloseCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "close",
		Short: "close the wallet",
		Run:   closec,
	}
	return cmd
}

func closec(cmd *cobra.Command, args []string) {
	_ = args
	o := cmd.Context["wallet"]
	w := o.(wallet.Wallet)
	w.Close()
}
