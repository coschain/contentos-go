package commands

import (
	"context"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet-cli/commands/utils"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
)

var TransferVestCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "transfer_vest",
		Short:   "convert COS to VEST",
		Long:    "convert amounts of liquidity COS to VEST",
		Example: "transfer_vest alice alice 500.000000 \"memo\"",
		Args:    cobra.ExactArgs(4),
		Run:     transferVest,
	}
	utils.ProcessEstimate(cmd)
	return cmd
}

func transferVest(cmd *cobra.Command, args []string) {
	defer func() {
		utils.EstimateStamina = false
	}()
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	mywallet := w.(wallet.Wallet)
	from := args[0]
	to := args[1]
	amount, err := utils.ParseCos(args[2])
	memo := args[3]
	if err != nil {
		fmt.Println(err)
		return
	}
	fromAccount, ok := mywallet.GetUnlockedAccount(from)
	if !ok {
		fmt.Println(fmt.Sprintf("account: %s should be unlocked or created first", from))
		return
	}

	transferv_op := &prototype.TransferToVestOperation{
		From:   &prototype.AccountName{Value: from},
		To:     &prototype.AccountName{Value: to},
		Amount: prototype.NewCoin(amount),
		Memo:	memo,
	}

	signTx, err := utils.GenerateSignedTxAndValidate(cmd, []interface{}{transferv_op}, fromAccount)
	if err != nil {
		fmt.Println(err)
		return
	}

	if utils.EstimateStamina {
		req := &grpcpb.EsimateRequest{Transaction:signTx}
		res,err := client.EstimateStamina(context.Background(), req)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(res.Invoice)
		}
	} else {
		req := &grpcpb.BroadcastTrxRequest{Transaction: signTx}
		resp, err := client.BroadcastTrx(context.Background(), req)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(fmt.Sprintf("Result: %v", resp))
		}
	}
}
