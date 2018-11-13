package commands

import (
	"context"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
)

var CreateCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "create a new account",
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
	name := args[0]
	passphrase := args[1]
	if err != nil {
		common.Fatalf(fmt.Sprintf("err: %v", err))
	}
	privKey, err := prototype.PrivateKeyFromWIF(privKeyStr)
	if err != nil {
		common.Fatalf(fmt.Sprintf("err: %v", err))
	}
	acop := &prototype.AccountCreateOperation{
		Fee:            &prototype.Coin{Amount: &prototype.Safe64{Value: 1}},
		Creator:        &prototype.AccountName{Value: name},
		NewAccountName: &prototype.AccountName{Value: name},
		Owner: &prototype.Authority{
			Cf:              prototype.Authority_active,
			WeightThreshold: 1,
			AccountAuths: []*prototype.KvAccountAuth{
				&prototype.KvAccountAuth{
					Name:   &prototype.AccountName{Value: "alice"},
					Weight: 3,
				},
			},
			KeyAuths: []*prototype.KvKeyAuth{
				&prototype.KvKeyAuth{
					Key: &prototype.PublicKeyType{
						Data: []byte{0},
					},
					Weight: 23,
				},
			},
		},
	}
	tx := &prototype.Transaction{RefBlockNum: 0, RefBlockPrefix: 0, Expiration: &prototype.TimePointSec{UtcSeconds: 0}}
	acopTrx := &prototype.Operation_Op1{}
	acopTrx.Op1 = acop
	tx.Operations = append(tx.Operations, &prototype.Operation{Op: acopTrx})
	signTx := prototype.SignedTransaction{Trx: tx}
	signTx.Sign(privKey, prototype.ChainId{Value: 0})
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
