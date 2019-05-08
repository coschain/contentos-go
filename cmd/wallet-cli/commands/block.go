package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/rpc/pb"
	"strconv"
)

var BlockCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use: "block",
	}

	blockCmd := &cobra.Command{
		Use:   "get",
		Short: "get block detail info",
		Example: "block get id",
		Args:  cobra.ExactArgs(1),
		Run:   blockQuery,
	}

	cmd.AddCommand(blockCmd)

	return cmd
}


func blockQuery(cmd *cobra.Command, args []string) {
	c := cmd.Context["rpcclient"]
	rpc := c.(grpcpb.ApiServiceClient)

	height, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		fmt.Println(err)
		return
	}

	req := &grpcpb.GetSignedBlockRequest{Start:height}
	resp, err := rpc.GetSignedBlock(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		buf, _ := json.MarshalIndent(resp, "", " ")
		//buf, _ := json.Marshal(resp)

		fmt.Println(fmt.Sprintf("block info: %s", buf))
	}
}