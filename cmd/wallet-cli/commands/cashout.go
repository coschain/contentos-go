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
		Args:  cobra.ExactArgs(2),
		Run: cashOut,
	}

	return cmd
}

func cashOut(cmd *cobra.Command, args []string) {
	//w := cmd.Context["wallet-cli"]
	//mywallet := w.(*wallet-cli.BaseWallet)
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