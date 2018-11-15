package commands

import (
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
)

var InfoCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "info",
		Short:   "display an unlocked account's info",
		Example: "info alice",
		Args:    cobra.ExactArgs(1),
		Run:     info,
	}
	return cmd
}

func info(cmd *cobra.Command, args []string) {
	o := cmd.Context["wallet"]
	w := o.(wallet.Wallet)
	name := args[0]
	content := w.Info(name)
	fmt.Println(content)
}
