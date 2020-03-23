package commands

import (
	"context"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet-cli/commands/utils"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
	"strconv"
)

var VoteCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "vote",
		Short:   "vote to a post",
		Example: "vote [voter] [postId]",
		Args:    cobra.ExactArgs(2),
		Run:     vote,
	}
	utils.ProcessEstimate(cmd)
	return cmd
}

func vote(cmd *cobra.Command, args []string) {
	defer func() {
		utils.EstimateStamina = false
	}()
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	mywallet := w.(wallet.Wallet)
	voter := args[0]
	idx, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		fmt.Println(err)
		return
	}
	voterAccount, ok := mywallet.GetUnlockedAccount(voter)
	if !ok {
		fmt.Println(fmt.Sprintf("account: %s should be unlocked or created first", voter))
		return
	}

	vote_op := &prototype.VoteOperation{
		Voter: &prototype.AccountName{Value: voter},
		Idx:   idx,
	}

	signTx, err := utils.GenerateSignedTxAndValidate(cmd, []interface{}{vote_op}, voterAccount)
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
