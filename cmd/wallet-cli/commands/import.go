package commands

import (
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet-cli/commands/utils"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
	"github.com/coschain/contentos-go/prototype"
)

var importForceFlag bool

var ImportCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "import",
		Short:   "import an account",
		Example: "import [name] [privkey]",
		Args:    cobra.ExactArgs(2),
		Run:     importAccount,
	}
	cmd.Flags().BoolVarP(&importForceFlag, "force", "f", false, "import --force")
	return cmd
}

func importAccount(cmd *cobra.Command, args []string) {
	//c := cmd.Context["rpcclient"]
	//client := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	r := cmd.Context["preader"]
	preader := r.(utils.PasswordReader)
	mywallet := w.(*wallet.BaseWallet)
	name := args[0]
	privKeyStr := args[1]
	passphrase, err := utils.GetPassphrase(preader)
	if err != nil {
		fmt.Println(err)
		return
	}
	if !importForceFlag {
		// just try to load or reload, if the name exist then we can find it in next step
		_ = mywallet.Load(name)
		ok := mywallet.IsExist(name)
		if ok {
			fmt.Println(fmt.Sprintf("the account: %s has been in your local keychain, please load it or you can import -f",
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
