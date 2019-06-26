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

var fundToContract uint64 = 0
var maxGas uint64 = 0

var CallCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "call",
		Short:   "call a deployed contract",
		Example: "call [caller] [owner] [contract_name] [method] [args]",
		Args:    cobra.ExactArgs(5),
		Run:     call,
	}
	cmd.Flags().Uint64VarP(&fundToContract, "fund", "f", 0, `call [caller] [owner] [contract_name] [args]  -f 300`)
	cmd.Flags().Uint64VarP(&maxGas, "gas", "g", 0, `call [caller] [owner] [contract_name] [args]  -g 300`)
	utils.ProcessEstimate(cmd)
	return cmd
}

func call(cmd *cobra.Command, args []string) {
	defer func() {
		utils.EstimateStamina = false
		fundToContract = 0
		maxGas = 0
	}()
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
	method := args[3]

	params := args[4]
	contractDeployOp := &prototype.ContractApplyOperation{
		Caller:   &prototype.AccountName{Value: caller},
		Owner:    &prototype.AccountName{Value: owner},
		Amount:   &prototype.Coin{Value: fundToContract},
		Contract: cname,
		Params:   params,
		Method:	  method,
	}
	signTx, err := utils.GenerateSignedTxAndValidate2(client, []interface{}{contractDeployOp}, acc)
	if err != nil {
		fmt.Println(err)
		return
	}

	if utils.EstimateStamina {
		req := &grpcpb.EsimateRequest{Transaction:signTx}
		res,err := client.EstimateStamina(context.Background(), req)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(res.Invoice)
		}
	} else {
		req := &grpcpb.BroadcastTrxRequest{Transaction: signTx}
		resp, err := client.BroadcastTrx(context.Background(), req)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(fmt.Sprintf("Result: %v", resp))
		}
	}
}
