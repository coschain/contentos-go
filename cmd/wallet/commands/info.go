package commands

import (
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet/wallet"
)

var InfoCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "display an account info",
		Run:   info,
	}
	return cmd
}

func info(cmd *cobra.Command, args []string) {
	o := cmd.Context["wallet"]
	w := o.(*wallet.BaseWallet)
	name := args[0]
	content := w.Info(name)
	fmt.Println(content)
}
