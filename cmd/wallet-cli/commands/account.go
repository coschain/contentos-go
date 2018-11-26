package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
)

var AccountCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use: "account",
	}

	getAccountCmd := &cobra.Command{
		Use:   "get",
		Short: "get account info",
		Args:  cobra.ExactArgs(1),
		Run:   getAccount,
	}

	cmd.AddCommand(getAccountCmd)

	return cmd
}

func getAccount(cmd *cobra.Command, args []string) {
	//w := cmd.Context["wallet-cli"]
	//mywallet := w.(*wallet-cli.BaseWallet)
	c := cmd.Context["rpcclient"]
	rpc := c.(grpcpb.ApiServiceClient)
	name := args[0]
	req := &grpcpb.GetAccountByNameRequest{AccountName: &prototype.AccountName{Value: name}}
	resp, err := rpc.GetAccountByName(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		buf, _ := json.Marshal(resp)
		fmt.Println(fmt.Sprintf("GetAccountByName detail: %s", buf))
	}
}

//func getFollowers(cmd *cobra.Command, args []string) {
//	c := cmd.Context["rpcclient"]
//	rpc := c.(grpcpb.ApiServiceClient)
//
//	name := args[0]
//	req := &grpcpb.GetFollowerListByNameRequest{AccountName: &prototype.AccountName{Value: name}}
//	resp, err := rpc.GetFollowerListByName(context.Background(), req)
//	if err != nil {
//		fmt.Println(err)
//	} else {
//		buf, _ := json.Marshal(resp)
//		fmt.Println(fmt.Sprintf("GetAccountByName detail: %s", string(buf)))
//	}
//}
