package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet-cli/commands/utils"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
	"strconv"
)

var ClaimCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use: "claim",
	}

	queryCmd := &cobra.Command{
		Use:     "query",
		Short:   "query account reward",
		Example: "claim query [account]",
		Args:    cobra.ExactArgs(1),
		Run:     queryClaim,
	}

	rewardCmd := &cobra.Command{
		Use:     "reward",
		Short:   "claim account reward",
		Example: "claim reward [account] [amount]",
		Args:    cobra.ExactArgs(2),
		Run:     rewardClaim,
	}

	cmd.AddCommand(queryCmd)
	cmd.AddCommand(rewardCmd)
	return cmd
}

var ClaimAllCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "claimAll",
		Short:   "claim account all reward",
		Example: "claimAll [account]",
		Args:    cobra.ExactArgs(1),
		Run:     rewardClaimAll,
	}
	return cmd
}

func queryClaim(cmd *cobra.Command, args []string) {
	c := cmd.Context["rpcclient"]
	rpc := c.(grpcpb.ApiServiceClient)
	name := args[0]
	// todo need get account reward by name
	req := &grpcpb.GetAccountRewardByNameRequest{AccountName: &prototype.AccountName{Value: name}}
	resp, err := rpc.GetAccountRewardByName(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		buf, _ := json.MarshalIndent(resp, "", "\t")
		fmt.Println(fmt.Sprintf("GetAccountRewardByName detail: %s", buf))
	}
}

func rewardClaim(cmd *cobra.Command, args []string) {
	c := cmd.Context["rpcclient"]
	rpc := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	mywallet := w.(wallet.Wallet)

	name := args[0]
	account, ok := mywallet.GetUnlockedAccount(name)

	if !ok {
		fmt.Println(fmt.Sprintf("account: %s should be loaded or created first", name))
		return
	}

	amount, err := strconv.ParseUint(args[1], 10, 64)

	if err != nil {
		fmt.Println(err)
		return
	}

	claimOp := &prototype.ClaimOperation{
		Account: &prototype.AccountName{Value: name},
		Amount:  amount,
	}

	signTx, err := utils.GenerateSignedTxAndValidate([]interface{}{claimOp}, account)
	if err != nil {
		fmt.Println(err)
		return
	}
	req := &grpcpb.BroadcastTrxRequest{Transaction: signTx}
	resp, err := rpc.BroadcastTrx(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(fmt.Sprintf("Result: %v", resp))
	}
}

func rewardClaimAll(cmd *cobra.Command, args []string) {
	c := cmd.Context["rpcclient"]
	rpc := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	mywallet := w.(wallet.Wallet)

	name := args[0]
	account, ok := mywallet.GetUnlockedAccount(name)

	if !ok {
		fmt.Println(fmt.Sprintf("account: %s should be loaded or created first", name))
		return
	}

	claimOp := &prototype.ClaimAllOperation{
		Account: &prototype.AccountName{Value: name},
	}

	signTx, err := utils.GenerateSignedTxAndValidate([]interface{}{claimOp}, account)
	if err != nil {
		fmt.Println(err)
		return
	}
	req := &grpcpb.BroadcastTrxRequest{Transaction: signTx}
	resp, err := rpc.BroadcastTrx(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(fmt.Sprintf("Result: %v", resp))
	}
}
