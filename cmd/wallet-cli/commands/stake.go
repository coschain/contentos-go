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

var StakeCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "stake",
		Short:   "stake some cos for stamina",
		Long:    "",
		Example: "stake alice bob 1.000000",
		Args:    cobra.MinimumNArgs(3),
		Run:     stake,
	}
	utils.ProcessEstimate(cmd)
	return cmd
}

func stake(cmd *cobra.Command, args []string) {
	defer func() {
		utils.EstimateStamina = false
	}()
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	mywallet := w.(wallet.Wallet)
	userFrom := args[0]
	userTo := args[1]
	amount, err := utils.ParseCos(args[2])
	if err != nil {
		fmt.Println(err)
		return
	}
	stakeAccount, ok := mywallet.GetUnlockedAccount(userFrom)
	if !ok {
		fmt.Println(fmt.Sprintf("account: %s should be loaded or created first", userFrom))
		return
	}

	stakeOp := &prototype.StakeOperation{
		From:   &prototype.AccountName{Value: userFrom},
		To:   &prototype.AccountName{Value: userTo},
		Amount:    prototype.NewCoin(amount),
	}

	signTx, err := utils.GenerateSignedTxAndValidate2(client, []interface{}{stakeOp}, stakeAccount)
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
