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

var VoteCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "vote",
		Short:   "vote to a post",
		Example: "vote [voter] [author] [permlink] [weight]",
		Args:    cobra.ExactArgs(4),
		Run:     vote,
	}
	return cmd
}

func vote(cmd *cobra.Command, args []string) {
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	mywallet := w.(*wallet.BaseWallet)
	voter := args[0]
	idx, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		fmt.Println(err)
		return
	}
	voterAccount, ok := mywallet.GetUnlockedAccount(voter)
	if !ok {
		fmt.Println(fmt.Sprintf("account: %s should be loaded or created first", voter))
		return
	}

	vote_op := &prototype.VoteOperation{
		Voter: &prototype.AccountName{Value: voter},
		Idx:   idx,
	}

	signTx, err := GenerateSignedTx([]interface{}{vote_op}, voterAccount)
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
