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
		Use:     "transfer",
		Short:   "transfer to another account",
		Long:    "transfer cos to another account by name, should unlock sender first",
		Example: "transfer alice bob 500",
		Args:    cobra.MinimumNArgs(3),
		Run:     transfer,
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
		Amount: prototype.NewCoin(uint64(amount)),
		Memo:   memo,
	}

	signTx, err := generateSignedTxAndValidate([]interface{}{transfer_op}, fromAccount)
	if err != nil {
		fmt.Println(err)
		return
	}
	req := &grpcpb.BroadcastTrxRequest{Transaction: signTx}
	resp, err := client.BroadcastTrx(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(fmt.Sprintf("Result: %v", resp))
	}

}
