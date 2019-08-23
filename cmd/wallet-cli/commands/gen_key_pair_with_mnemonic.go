package commands

import (
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
)

var GenKeyPairWithMnemonicCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "genKeyPair",
		Short: "generate new key pair and mnemonic",
		Run:   genKeyPairAndMnemonic,
	}
	return cmd
}

func genKeyPairAndMnemonic(cmd *cobra.Command, args []string) {

	w := cmd.Context["wallet"]
	mywallet := w.(*wallet.BaseHDWallet)
	mnemonic, err := mywallet.GenerateNewMnemonic()
	if err != nil {
		fmt.Println("Generate New Key Error:", err)
	} else {
		pubKeyStr, privKeyStr, err := mywallet.GenerateFromMnemonic(mnemonic)
		if err != nil {
			fmt.Println("Generate New Key Error:", err)
		}
		fmt.Println("Mnemonic:", mnemonic)
		fmt.Println("Public  Key: ", pubKeyStr)
		fmt.Println("Private Key: ", privKeyStr)
	}
}