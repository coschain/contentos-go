package commands

import (
	"context"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet-cli/commands/utils"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
	"strings"
)

var postBeneficiaryRoute map[string]int

var PostCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "post",
		Short:   "post a topic",
		Example: "post [author] [tags] [title] [content]",
		Args:    cobra.ExactArgs(4),
		Run:     post,
	}
	cmd.Flags().StringToIntVarP(&postBeneficiaryRoute, "beneficiary", "b", map[string]int{},
		`post --beneficiary="Alice=5,Bob=10"`)
	return cmd
}

func post(cmd *cobra.Command, args []string) {
	defer func() {
		postBeneficiaryRoute = map[string]int{}
	}()
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	mywallet := w.(wallet.Wallet)
	author := args[0]
	authorAccount, ok := mywallet.GetUnlockedAccount(author)
	if !ok {
		fmt.Println(fmt.Sprintf("account: %s should be loaded or created first", author))
		return
	}
	tagsStr := args[1]
	if len(tagsStr) == 0 {
		fmt.Println("tags cannot be empty string. It should be a sequence of words split by comma")
		return
	}
	tags := strings.Split(tagsStr, ",")
	title := args[2]
	content := args[3]
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
	uuid := utils.GenerateUUID(author + title)
	post_op := &prototype.PostOperation{
		Uuid:          uuid,
		Owner:         &prototype.AccountName{Value: author},
		Title:         title,
		Content:       content,
		Tags:          tags,
		Beneficiaries: beneficiaries,
	}
	signTx, err := utils.GenerateSignedTxAndValidate(cmd, []interface{}{post_op}, authorAccount)
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
			fmt.Println(fmt.Sprintf("PostId: %d", uuid))
			fmt.Println(fmt.Sprintf("Result: %v", resp))
		}
	}
}
