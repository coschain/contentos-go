package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
	"strings"
)

var FollowCntCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "follow_count",
		Short:   "get account follow relation",
		Example: "follow_count [account_name]",
		Args:    cobra.ExactArgs(1),
		Run:     followCnt,
	}

	return cmd
}

func followCnt(cmd *cobra.Command, args []string) {
	c := cmd.Context["rpcclient"].(grpcpb.ApiServiceClient)
	//w := cmd.Context["wallet"].(*wallet.BaseWallet)

	name := strings.TrimSpace(args[0])
	if name != "" {
		req := &grpcpb.GetFollowCountByNameRequest{AccountName: &prototype.AccountName{Value: name}}
		resp, err := c.GetFollowCountByName(context.Background(), req)
		if err != nil {
			fmt.Println(err)
		} else {
			buf, err := json.Marshal(resp)
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(fmt.Sprintf("follow_count result: [%s]", buf))
			}
		}
	} else {
		fmt.Println("follow_count result: []")
	}
}
