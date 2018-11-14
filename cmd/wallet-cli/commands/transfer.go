package commands

import (
	"context"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
	"strconv"
)

var TransferCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer",
		Short: "transfer to another account",
		Args:  cobra.MinimumNArgs(3),
		Run:   transfer,
	}
	return cmd
}

func transfer(cmd *cobra.Command, args []string) {
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	mywallet := w.(*wallet.BaseWallet)
	from := args[0]
	to := args[1]
	amount, err := strconv.ParseInt(args[2], 10, 64)
	if err != nil {
		fmt.Println(err)
		return
	}
	memo := ""
	if len(args) > 3 {
		memo = args[3]
	}
	fromAccount, ok := mywallet.GetUnlockedAccount(from)
	if !ok {
		fmt.Println(fmt.Sprintf("account: %s should be loaded or created first", from))
		return
	}

	transfer_op := &prototype.TransferOperation{
		From:   &prototype.AccountName{Value: from},
		To:     &prototype.AccountName{Value: to},
		Amount: &prototype.Coin{Amount: &prototype.Safe64{Value: amount}},
		Memo:   memo,
	}
	tx := &prototype.Transaction{RefBlockNum: 0, RefBlockPrefix: 0, Expiration: &prototype.TimePointSec{UtcSeconds: 0}}
	tx.AddOperation(transfer_op)
	signTx := prototype.SignedTransaction{Trx: tx}
	cid := prototype.ChainId{Value: 0}
	PrivateKey, err := prototype.PrivateKeyFromWIF(fromAccount.PrivKey)
	if err != nil {
		fmt.Println(PrivateKey)
		return
	}
	res := signTx.Sign(PrivateKey, cid)
	signTx.Signatures = append(signTx.Signatures, &prototype.SignatureType{Sig: res})
	req := &grpcpb.BroadcastTrxRequest{Transaction: &signTx}
	resp, err := client.BroadcastTrx(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(fmt.Sprintf("Result: %v", resp))
	}

}
