package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
	"strconv"
)

var CashoutCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use: "cashout",
	}

	accountCmd := &cobra.Command{
		Use: "account",
		Short: "get account reward in block",
		Args: cobra.ExactArgs(2),
		Run: cashout,
	}

	blockCmd := &cobra.Command{
		Use:   "block",
		Short: "get accounts info in block",
		Args:  cobra.ExactArgs(1),
		Run:   cashoutBlock,
	}

	cmd.AddCommand(accountCmd)
	cmd.AddCommand(blockCmd)

	return cmd
}

func cashout(cmd *cobra.Command, args []string) {
	c := cmd.Context["rpcclient"]
	rpc := c.(grpcpb.ApiServiceClient)
	name := args[0]

	height, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		fmt.Println(err)
		return
	}

	req := &grpcpb.GetAccountCashoutRequest{AccountName: &prototype.AccountName{Value: name},BlockHeight:height}
	resp, err := rpc.GetAccountCashout(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		buf, _ := json.MarshalIndent(resp, "", "\t")
		fmt.Println(fmt.Sprintf("GetAccountCashout detail: %s", buf))
	}
}

func cashoutBlock(cmd *cobra.Command, args []string) {
	c := cmd.Context["rpcclient"]
	rpc := c.(grpcpb.ApiServiceClient)

	height, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		fmt.Println(err)
		return
	}

	req := &grpcpb.GetBlockCashoutRequest{BlockHeight:height}
	resp, err := rpc.GetBlockCashout(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		buf, _ := json.MarshalIndent(resp, "", "\t")
		fmt.Println(fmt.Sprintf("GetAccountCashout detail: %s", buf))
	}
}