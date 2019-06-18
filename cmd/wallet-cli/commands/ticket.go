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

var TicketCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use: "ticket",
	}

	acquireCmd := &cobra.Command{
		Use:   "acquire",
		Short: "acquire tickets using vesting",
		Example: "ticket acquire [name] [count]",
		Args:  cobra.ExactArgs(2),
		Run:   acquireTicket,
	}

	voteCmd := &cobra.Command{
		Use: "vote",
		Short: "vote tickets to post",
		Example: "ticket vote [name] [postId] [count]",
		Args: cobra.ExactArgs(3),
		Run: voteByTicket,
	}

	cmd.AddCommand(acquireCmd)
	cmd.AddCommand(voteCmd)

	return cmd
}

func acquireTicket(cmd *cobra.Command, args []string) {
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	mywallet := w.(wallet.Wallet)
	name := args[0]
	count, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		fmt.Println()
	}
	account, ok := mywallet.GetUnlockedAccount(name)
	if !ok {
		fmt.Println(fmt.Sprintf("account: %s should be loaded or created first", name))
		return
	}
	acquireTicketOp := &prototype.AcquireTicketOperation{
		Account: &prototype.AccountName{Value:name},
		Count: count,
	}
	signTx, err := utils.GenerateSignedTxAndValidate2(client, []interface{}{acquireTicketOp}, account)
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

func voteByTicket(cmd *cobra.Command, args []string) {
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	mywallet := w.(wallet.Wallet)
	name := args[0]
	postId, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		fmt.Println(err)
	}
	count, err := strconv.ParseUint(args[2], 10, 64)
	if err != nil {
		fmt.Println()
	}
	account, ok := mywallet.GetUnlockedAccount(name)
	if !ok {
		fmt.Println(fmt.Sprintf("account: %s should be loaded or created first", name))
		return
	}
	voteByTicketOp := &prototype.VoteByTicketOperation{
		Account: &prototype.AccountName{Value:name},
		Idx: postId,
		Count: count,
	}
	signTx, err := utils.GenerateSignedTxAndValidate2(client, []interface{}{voteByTicketOp}, account)
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


