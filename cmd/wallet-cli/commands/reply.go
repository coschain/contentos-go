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

var replyBeneficiaryRoute map[string]int

var ReplyCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "reply",
		Short:   "reply a topic",
		Example: "reply [author] [content] [postId]",
		Args:    cobra.ExactArgs(3),
		Run:     reply,
	}
	cmd.Flags().StringToIntVarP(&replyBeneficiaryRoute, "beneficiary", "b", map[string]int{},
		`reply --beneficiary="Alice=5,Bob=10"`)
	return cmd
}

func reply(cmd *cobra.Command, args []string) {
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	mywallet := w.(*wallet.BaseWallet)
	author := args[0]
	authorAccount, ok := mywallet.GetUnlockedAccount(author)
	if !ok {
		fmt.Println(fmt.Sprintf("account: %s should be loaded or created first", author))
		return
	}
	content := args[1]
	postId, err := strconv.ParseUint(args[2], 10, 64)
	if err != nil {
		fmt.Println(err)
		return
	}
	beneficiaries := []*prototype.BeneficiaryRouteType{}
	accumulateWeight := 0
	for k, v := range postBeneficiaryRoute {
		if v < 0 {
			fmt.Println("weight should greater than zero")
			return
		}

		if v > 10 {
			fmt.Println("either beneficiary route should not greater than 10%")
			return
		}

		if accumulateWeight > 10 {
			fmt.Println("accumulated weight should not greater than 10%")
			return
		}

		accumulateWeight += v
		route := &prototype.BeneficiaryRouteType{
			Name:   &prototype.AccountName{Value: k},
			Weight: uint32(v),
		}

		beneficiaries = append(beneficiaries, route)
	}
	uuid := utils.GenerateUUID(author)
	reply_op := &prototype.ReplyOperation{
		Uuid:          uuid,
		Owner:         &prototype.AccountName{Value: author},
		Content:       content,
		ParentUuid:    postId,
		Beneficiaries: beneficiaries,
	}
	signTx, err := utils.GenerateSignedTxAndValidate([]interface{}{reply_op}, authorAccount)
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
