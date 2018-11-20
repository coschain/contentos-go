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

var TransferVestingCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "transfer_vesting",
		Short:   "convert cos to vesting",
		Long:    "convert amounts of liquidity cos to vesting",
		Example: "transfer_vesting alice alice 500",
		Args:    cobra.ExactArgs(3),
		Run:     transferVesting,
	}
	return cmd
}

func transferVesting(cmd *cobra.Command, args []string) {
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
	fromAccount, ok := mywallet.GetUnlockedAccount(from)
	if !ok {
		fmt.Println(fmt.Sprintf("account: %s should be loaded or created first", from))
		return
	}

	transferv_op := &prototype.TransferToVestingOperation{
		From:   &prototype.AccountName{Value: from},
		To:     &prototype.AccountName{Value: to},
		Amount: prototype.NewCoin(uint64(amount)),
	}

	signTx, err := generateSignedTxAndValidate([]interface{}{transferv_op}, fromAccount)
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
