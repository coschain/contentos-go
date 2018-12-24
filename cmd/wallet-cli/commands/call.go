package commands

import (
	"context"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet-cli/commands/utils"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
)

var CallCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "call",
		Short:   "call a deployed contract",
		Example: "deploy [caller] [owner] [contract_name] [args]",
		Args:    cobra.ExactArgs(4),
		Run:     call,
	}
	return cmd
}

func call(cmd *cobra.Command, args []string) {
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	mywallet := w.(wallet.Wallet)
	caller := args[0]
	acc, ok := mywallet.GetUnlockedAccount(caller)
	if !ok {
		fmt.Println(fmt.Sprintf("caller: %s should be loaded or created first", caller))
		return
	}
	owner := args[1]
	cname := args[2]
	params := args[3]
	contractDeployOp := &prototype.ContractApplyOperation{
		Caller:   &prototype.AccountName{Value: caller},
		Owner:    &prototype.AccountName{Value: owner},
		Contract: cname,
		Params:   params,
	}
	signTx, err := utils.GenerateSignedTxAndValidate([]interface{}{contractDeployOp}, acc)
	if err != nil {
		fmt.Println(err)
		return
	}
	req := &grpcpb.BroadcastTrxRequest{Transaction: signTx}
	resp, err := client.BroadcastTrx(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(fmt.Sprintf("Result: %v", resp))
	}

}
