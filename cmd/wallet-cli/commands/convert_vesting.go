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

var ConvertVestCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "convert_vest",
		Short:   "convert VEST to COS",
		Long:    "convert VEST to COS, it takes 13 weeks before finish",
		Example: "convert_vest alice 500.000000",
		Args:    cobra.MinimumNArgs(2),
		Run:     convert,
	}
	utils.ProcessEstimate(cmd)
	return cmd
}

func convert(cmd *cobra.Command, args []string) {
	defer func() {
		utils.EstimateStamina = false
	}()
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	mywallet := w.(wallet.Wallet)
	from := args[0]
	amount, err := utils.ParseCos(args[1])
	if err != nil {
		fmt.Println(err)
		return
	}
	fromAccount, ok := mywallet.GetUnlockedAccount(from)
	if !ok {
		fmt.Println(fmt.Sprintf("account: %s should be unlocked or created first", from))
		return
	}

	convert_vest_op := &prototype.ConvertVestOperation{
		From:   &prototype.AccountName{Value: from},
		Amount: prototype.NewVest(uint64(amount)),
	}

	signTx, err := utils.GenerateSignedTxAndValidate(cmd, []interface{}{convert_vest_op}, fromAccount)
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
