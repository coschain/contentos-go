package commands

import (
	"context"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet-cli/commands/utils"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
)

var createAccountFee uint64

var CreateCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create",
		Short:   "create a new account",
		Example: "create [creator] [name]",
		Args:    cobra.ExactArgs(2),
		Run:     create,
	}

	cmd.Flags().Uint64VarP(&createAccountFee, "fee", "f", constants.DefaultAccountCreateFee, `create alice bob --fee 1`)

	return cmd
}

func create(cmd *cobra.Command, args []string) {
	defer func() {
		createAccountFee = constants.DefaultAccountCreateFee
	}()
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	mywallet := w.(wallet.Wallet)
	r := cmd.Context["preader"]
	preader := r.(utils.PasswordReader)
	creator := args[0]
	creatorAccount, ok := mywallet.GetUnlockedAccount(creator)
	if !ok {
		fmt.Println(fmt.Sprintf("creator: %s should be loaded or created first", creator))
		return
	}
	pubKeyStr, privKeyStr, err := mywallet.GenerateNewKey()
	pubkey, _ := prototype.PublicKeyFromWIF(pubKeyStr)
	name := args[1]
	passphrase, err := utils.GetPassphrase(preader)
	if err != nil {
		fmt.Println(err)
		return
	}

	acop := &prototype.AccountCreateOperation{
		Fee:            prototype.NewCoin(createAccountFee),
		Creator:        &prototype.AccountName{Value: creator},
		NewAccountName: &prototype.AccountName{Value: name},
		Owner:          pubkey,
	}
	signTx, err := utils.GenerateSignedTxAndValidate2(client, []interface{}{acop}, creatorAccount)
	if err != nil {
		fmt.Println(err)
		return
	}
	req := &grpcpb.BroadcastTrxRequest{Transaction: signTx}
	resp, err := client.BroadcastTrx(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		if resp.Invoice.Status == 200 {
			err = mywallet.Create(name, passphrase, pubKeyStr, privKeyStr)
			if err != nil {
				fmt.Println(err)
			}
		}
		fmt.Println(fmt.Sprintf("Result: %v", resp))
	}
}
