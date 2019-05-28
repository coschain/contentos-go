package commands

import (
	"context"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet-cli/commands/utils"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
	"strconv"
)

var StakeCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "stake",
		Short:   "stake some cos for stamina",
		Long:    "",
		Example: "stake alice 500",
		Args:    cobra.MinimumNArgs(2),
		Run:     stake,
	}
	return cmd
}

func stake(cmd *cobra.Command, args []string) {
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	mywallet := w.(wallet.Wallet)
	user := args[0]
	amount, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		fmt.Println(err)
		return
	}
	stakeAccount, ok := mywallet.GetUnlockedAccount(user)
	if !ok {
		fmt.Println(fmt.Sprintf("account: %s should be loaded or created first", user))
		return
	}

	stakeOp := &prototype.StakeOperation{
		Account:   &prototype.AccountName{Value: user},
		Amount:    prototype.NewCoin(uint64(amount)),
	}

	signTx, err := utils.GenerateSignedTxAndValidate2(client, []interface{}{stakeOp}, stakeAccount)
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
