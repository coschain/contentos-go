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

var UnStakeCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "unstake",
		Short:   "unstake some cos for stamina",
		Long:    "",
		Example: "unstake alice 500.000000",
		Args:    cobra.MinimumNArgs(3),
		Run:     unstake,
	}
	utils.ProcessEstimate(cmd)
	return cmd
}

func unstake(cmd *cobra.Command, args []string) {
	defer func() {
		utils.EstimateStamina = false
	}()
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	mywallet := w.(wallet.Wallet)
	userCreditor := args[0]
	userDebtor := args[1]
	amount, err := utils.ParseCos(args[2])
	if err != nil {
		fmt.Println(err)
		return
	}
	stakeAccount, ok := mywallet.GetUnlockedAccount(userCreditor)
	if !ok {
		fmt.Println(fmt.Sprintf("account: %s should be loaded or created first", userCreditor))
		return
	}

	unStakeOp := &prototype.UnStakeOperation{
		Creditor:   &prototype.AccountName{Value: userCreditor},
		Debtor:   &prototype.AccountName{Value: userDebtor},
		Amount:    prototype.NewCoin(amount),
	}

	signTx, err := utils.GenerateSignedTxAndValidate(cmd, []interface{}{unStakeOp}, stakeAccount)
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
