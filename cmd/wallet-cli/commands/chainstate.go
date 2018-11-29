package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/rpc/pb"
)

var ChainStateCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chainstate",
		Short: "get chainstate info",
		Run:   getChainState,
	}

	return cmd
}

func getChainState(cmd *cobra.Command, args []string) {

	c := cmd.Context["rpcclient"]
	rpc := c.(grpcpb.ApiServiceClient)
	req := &grpcpb.NonParamsRequest{}
	resp, err := rpc.GetChainState(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		buf, _ := json.MarshalIndent(resp, "", "\t")
		fmt.Println(fmt.Sprintf("GetChainState detail: %s", buf))
	}
}
