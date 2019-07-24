package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
)

var limit uint32

var StakerListCmd = func() *cobra.Command {

	cmd := &cobra.Command{
		Use:     "stakers",
	}

	getMyStakersCmd := &cobra.Command {
		Use:     "forme",
		Short:   "query who stake to me",
		Example: "forme bob alice mike",
		Args:    cobra.MinimumNArgs(3),
		Run:     stakersForMe,
	}
	getMyStakersCmd.Flags().Uint32VarP(&limit, "limit", "", 30, `stakers forme initminer accountstart accountend --limit 10`)

	getMyStakesCmd := &cobra.Command {
		Use:     "toother",
		Short:   "query users that i stake to",
		Example: "forother bob alice mike",
		Args:    cobra.MinimumNArgs(3),
		Run:     stakersToOther,
	}
	getMyStakesCmd.Flags().Uint32VarP(&limit, "limit", "", 30, `stakers forme initminer accountstart accountend --limit 10`)


	cmd.AddCommand(getMyStakersCmd)
	cmd.AddCommand(getMyStakesCmd)
	return cmd
}

func stakersForMe(cmd *cobra.Command, args []string) {
	defer func() {
		limit = 30
	}()
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)

	userTo := args[0]
	userStart := args[1]
	userEnd := args[2]

	req := &grpcpb.GetMyStakerListByNameRequest{
		Limit:limit,
		Start:&prototype.StakeRecordReverse{To:prototype.NewAccountName(userTo),From:prototype.NewAccountName(userStart),},
		End:&prototype.StakeRecordReverse{To:prototype.NewAccountName(userTo),From:prototype.NewAccountName(userEnd),},
	}

	resp, err := client.GetMyStakers(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		buf, _ := json.MarshalIndent(resp, "", "\t")
		fmt.Println(fmt.Sprintf("stake to %s detail: %s", userTo,buf))
	}
}

func stakersToOther(cmd *cobra.Command, args []string) {
	defer func() {
		limit = 30
	}()
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)

	userFrom := args[0]
	userStart := args[1]
	userEnd := args[2]

	req := &grpcpb.GetMyStakeListByNameRequest{
		Limit:limit,
		Start:&prototype.StakeRecord{From:prototype.NewAccountName(userFrom),To:prototype.NewAccountName(userStart),},
		End:&prototype.StakeRecord{From:prototype.NewAccountName(userFrom),To:prototype.NewAccountName(userEnd),},
	}

	resp, err := client.GetMyStakes(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		buf, _ := json.MarshalIndent(resp, "", "\t")
		fmt.Println(fmt.Sprintf("%s stake to others detail: %s",userFrom, buf))
	}
}
