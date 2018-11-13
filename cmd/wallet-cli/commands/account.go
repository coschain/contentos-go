package commands

import (
	"context"
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
		Run:   getAccount,
	}

	cmd.AddCommand(getAccountCmd)

	return cmd
}

func getAccount(cmd *cobra.Command, args []string) {
	//w := cmd.Context["wallet-cli"]
	//mywallet := w.(*wallet-cli.BaseWallet)
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	name := args[0]
	req := &grpcpb.GetAccountByNameRequest{AccountName: &prototype.AccountName{Value: name}}
	resp, err := client.GetAccountByName(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(fmt.Sprintf("GetAccountByName detail: %s", resp.AccountName))
	}
}
