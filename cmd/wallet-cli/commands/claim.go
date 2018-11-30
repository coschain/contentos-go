package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
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
		Example: "claim reward [account]",
		Args:    cobra.ExactArgs(1),
		Run:     rewardClaim,
	}

	cmd.AddCommand(queryCmd)
	cmd.AddCommand(rewardCmd)
	return cmd
}

func queryClaim(cmd *cobra.Command, args []string) {
	c := cmd.Context["rpcclient"]
	rpc := c.(grpcpb.ApiServiceClient)
	name := args[0]
	// todo need get account reward by name
	req := &grpcpb.GetAccountByNameRequest{AccountName: &prototype.AccountName{Value: name}}
	resp, err := rpc.GetAccountByName(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		buf, _ := json.MarshalIndent(resp, "", "\t")
		fmt.Println(fmt.Sprintf("GetAccountByName detail: %s", buf))
	}
}

func rewardClaim(cmd *cobra.Command, args []string) {

}
