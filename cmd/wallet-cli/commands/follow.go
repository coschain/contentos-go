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

var followCancel bool

var FollowCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "follow",
		Short:   "follow an author",
		Example: "follow [follower] [following]",
		Args:    cobra.ExactArgs(2),
		Run:     follow,
	}

	cmd.Flags().BoolVarP(&followCancel, "cancel", "c", false, `follow alice bob --cancel`)
	utils.ProcessEstimate(cmd)
	return cmd
}

func follow(cmd *cobra.Command, args []string) {
	defer func() {
		utils.EstimateStamina = false
		followCancel = false
	}()
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	mywallet := w.(wallet.Wallet)
	follower := args[0]
	followerAccount, ok := mywallet.GetUnlockedAccount(follower)
	if !ok {
		fmt.Println(fmt.Sprintf("account: %s should be loaded or created first", follower))
		return
	}
	following := args[1]
	follow_op := &prototype.FollowOperation{
		Account:  &prototype.AccountName{Value: follower},
		FAccount: &prototype.AccountName{Value: following},
		Cancel:   followCancel,
	}

	signTx, err := utils.GenerateSignedTxAndValidate2(client, []interface{}{follow_op}, followerAccount)
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
