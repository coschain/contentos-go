package commands

import (
	"context"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
)

var CreateCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "create a new account",
		Args:  cobra.ExactArgs(3),
		Run:   create,
	}
	return cmd
}

func create(cmd *cobra.Command, args []string) {
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	mywallet := w.(*wallet.BaseWallet)
	pubKeyStr, privKeyStr, err := mywallet.GenerateNewKey()
	creator := args[0]
	creatorAccount, ok := mywallet.GetUnlockedAccount(creator)
	if !ok {
		fmt.Println(fmt.Sprintf("creator: %s should be loaded or created first", creator))
		return
	}
	name := args[1]
	passphrase := args[2]
	if err != nil {
		fmt.Println(err)
		return
	}
	creatorPrivKey, err := prototype.PrivateKeyFromWIF(creatorAccount.PrivKey)
	if err != nil {
		fmt.Println(err)
		return
	}
	acop := &prototype.AccountCreateOperation{
		Fee:            &prototype.Coin{Amount: &prototype.Safe64{Value: 1}},
		Creator:        &prototype.AccountName{Value: creator},
		NewAccountName: &prototype.AccountName{Value: name},
		Owner: &prototype.Authority{
			Cf:              prototype.Authority_active,
			WeightThreshold: 1,
			AccountAuths: []*prototype.KvAccountAuth{
				{
					Name:   &prototype.AccountName{Value: name},
					Weight: 3,
				},
			},
			KeyAuths: []*prototype.KvKeyAuth{
				{
					Key: &prototype.PublicKeyType{
						Data: []byte(pubKeyStr),
					},
					Weight: 23,
				},
			},
		},
	}
	tx := &prototype.Transaction{RefBlockNum: 0, RefBlockPrefix: 0, Expiration: &prototype.TimePointSec{UtcSeconds: 0}}
	tx.AddOperation(acop)
	signTx := prototype.SignedTransaction{Trx: tx}
	res := signTx.Sign(creatorPrivKey, prototype.ChainId{Value: 0})
	signTx.Signatures = append(signTx.Signatures, &prototype.SignatureType{Sig: res})
	req := &grpcpb.BroadcastTrxRequest{Transaction: &signTx}
	resp, err := client.BroadcastTrx(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		err = mywallet.Create(name, passphrase, pubKeyStr, privKeyStr)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(fmt.Sprintf("Result: %v", resp))
	}
}
