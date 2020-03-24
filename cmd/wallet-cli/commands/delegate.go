package commands

import (
	"context"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet-cli/commands/utils"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
	"strconv"
	"strings"
)

var delegationListOptionTo bool

var DelegateCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use: "delegate",
		Short: "vest delegation, claiming and order list",
	}

	listCmd := &cobra.Command{
		Use: "list",
		Short: "list delegation orders of specific account",
		Example: "delegate list alice",
		Args: cobra.ExactArgs(1),
		Run: listDelegations,
	}
	listCmd.Flags().BoolVarP(&delegationListOptionTo, "to", "t", false, `delegate list alice --to`)

	newCmd := &cobra.Command{
		Use: "new",
		Short: "create a new delegation order",
		Example: "delegate new alice bob 100.000000 86400",
		Args: cobra.ExactArgs(4),
		Run: newDelegation,
	}

	claimCmd := &cobra.Command{
		Use: "claim",
		Short: "claim the vests from a matured delegation order",
		Example: "delegate claim alice [order_id]",
		Args: cobra.ExactArgs(2),
		Run: claimDelegation,
	}

	cmd.AddCommand(listCmd)
	cmd.AddCommand(newCmd)
	cmd.AddCommand(claimCmd)
	utils.ProcessEstimate(cmd)
	return cmd
}

func listDelegations(cmd *cobra.Command, args []string) {
	defer func() {
		delegationListOptionTo = false
		utils.EstimateStamina = false
	}()
	const pageSize = 50
	const maxOrders = 1000
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	req := &grpcpb.GetVestDelegationOrderListRequest{
		Account: prototype.NewAccountName(args[0]),
		IsFrom: !delegationListOptionTo,
		Limit: pageSize,
		LastOrderId: 0,
	}
	var orders []*grpcpb.VestDelegationOrder
	for len(orders) < maxOrders {
		if resp, err := client.GetVestDelegationOrderList(context.Background(), req); err == nil {
			if resp == nil || len(resp.GetOrders()) == 0 {
				break
			}
			req.LastOrderId = resp.Orders[len(resp.Orders) - 1].Id
			orders = append(orders, resp.Orders...)
		} else {
			fmt.Println(err)
			return
		}
	}
	if len(orders) == 0 {
		fmt.Println("no orders found.")
		return
	}
	headBlock := uint64(0)
	if chainStateResp, err := client.GetChainState(context.Background(), new(grpcpb.NonParamsRequest)); err == nil {
		headBlock = chainStateResp.GetState().GetDgpo().GetHeadBlockNumber()
	}
	if len(orders) > maxOrders {
		orders = orders[:maxOrders]
		fmt.Printf("WARNING: Only %d orders are displayed.\n", maxOrders)
	}
	fmt.Printf("%10s %16s %16s %18s %10s %10s %8s %10s\n",
		"Order-ID", "From", "To", "Vests", "Created", "Maturity", "Matured", "Delivery")
	fmt.Println(strings.Repeat("-", 106))
	for _, order := range orders {
		delivery := "-"
		if order.Delivering {
			delivery = strconv.FormatUint(order.DeliveryBlock, 10)
		}
		matured := "no"
		if headBlock > order.MaturityBlock {
			matured = "yes"
		}
		fmt.Printf("%10d %16s %16s %11d.%06d %10d %10d %8s %10s\n",
			order.Id,
			order.FromAccount.Value,
			order.ToAccount.Value,
			order.Amount.Value / constants.COSTokenDecimals,
			order.Amount.Value % constants.COSTokenDecimals,
			order.CreatedBlock,
			order.MaturityBlock,
			matured,
			delivery,
		)
	}
}

func newDelegation(cmd *cobra.Command, args []string) {
	defer func() {
		utils.EstimateStamina = false
	}()
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	mywallet := w.(wallet.Wallet)

	fromAccountName := args[0]
	toAccountName := args[1]
	amount, err := utils.ParseCos(args[2])
	if err != nil {
		fmt.Println(err)
		return
	}
	expiration, err := strconv.ParseUint(args[3], 10, 64)
	if err != nil {
		fmt.Println(err)
		return
	}
	fromAccount, ok := mywallet.GetUnlockedAccount(fromAccountName)
	if !ok {
		fmt.Println(fmt.Sprintf("account: %s should be unlocked or created first", fromAccountName))
		return
	}
	op := &prototype.DelegateVestOperation{
		From: &prototype.AccountName{Value: fromAccountName},
		To: &prototype.AccountName{Value: toAccountName},
		Amount: &prototype.Vest{Value: amount},
		Expiration: expiration,
	}
	signTx, err := utils.GenerateSignedTxAndValidate(cmd, []interface{}{op}, fromAccount)
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

func claimDelegation(cmd *cobra.Command, args []string) {
	defer func() {
		utils.EstimateStamina = false
	}()
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	mywallet := w.(wallet.Wallet)

	accountName := args[0]
	orderId, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		fmt.Println(err)
		return
	}
	account, ok := mywallet.GetUnlockedAccount(accountName)
	if !ok {
		fmt.Println(fmt.Sprintf("account: %s should be unlocked or created first", accountName))
		return
	}
	op := &prototype.UnDelegateVestOperation{
		Account: &prototype.AccountName{Value: accountName},
		OrderId: orderId,
	}
	signTx, err := utils.GenerateSignedTxAndValidate(cmd, []interface{}{op}, account)
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
