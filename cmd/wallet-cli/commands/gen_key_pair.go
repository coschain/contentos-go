package commands

import (
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
)

var GenKeyPairCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "genKeyPair",
		Short: "generate new key pair",
		Run:   genKeyPair,
	}
	return cmd
}

func genKeyPair(cmd *cobra.Command, args []string) {

	w := cmd.Context["wallet"]
	mywallet := w.(*wallet.BaseWallet)
	pubKeyStr, privKeyStr, err := mywallet.GenerateNewKey()
	if err != nil {
		fmt.Println("Generate New Key Error:", err)
	} else {
		fmt.Println("Public  Key: ", pubKeyStr)
		fmt.Println("Private Key: ", privKeyStr)
	}

}
