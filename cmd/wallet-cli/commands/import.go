package commands

import (
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
	"github.com/coschain/contentos-go/prototype"
)

var importForceFlag bool

var ImportCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "import an account",

		Args: cobra.ExactArgs(3),
		Run:  importAccount,
	}
	cmd.Flags().BoolVarP(&importForceFlag, "force", "f", false, "import --force")
	return cmd
}

func importAccount(cmd *cobra.Command, args []string) {
	//c := cmd.Context["rpcclient"]
	//client := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	mywallet := w.(*wallet.BaseWallet)
	name := args[0]
	privKeyStr := args[1]
	passphrase := args[2]
	if !importForceFlag {
		err := mywallet.Load(name)
		if err != nil {
			fmt.Println(err)
			return
		}
		ok := mywallet.IsExist(name)
		if ok {
			fmt.Println(fmt.Sprintf("the account: %s has been in your local keychain, please load it or import -f",
				name))
			return
		}
	}
	privKey, err := prototype.PrivateKeyFromWIF(privKeyStr)
	if err != nil {
		fmt.Println(err)
		return
	}
	pubKey, err := privKey.PubKey()
	if err != nil {
		fmt.Println(err)
		return
	}
	pubKeyStr := pubKey.ToWIF()
	// fixme
	// the pubkey and account name should be check by api
	err = mywallet.Create(name, passphrase, pubKeyStr, privKeyStr)
	if err != nil {
		fmt.Println(err)
	}
}
