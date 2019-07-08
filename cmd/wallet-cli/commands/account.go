package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet-cli/commands/utils"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
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

	updateAccountCmd := &cobra.Command{
		Use:   "update",
		Short: "update account public key",
		Example: "account update [name] [newpubkey] [newprikey]",
		Args:  cobra.ExactArgs(3),
		Run:   updateAccount,
	}

	cmd.AddCommand(getAccountCmd)
	cmd.AddCommand(updateAccountCmd)
	utils.ProcessEstimate(cmd)

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
		buf, _ := json.MarshalIndent(resp, "", "\t")
		fmt.Println(fmt.Sprintf("GetAccountByName detail: %s", buf))
	}
}

func updateAccount(cmd *cobra.Command, args []string) {
	defer func() {
		utils.EstimateStamina = false
	}()
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	mywallet := w.(wallet.Wallet)
	r := cmd.Context["preader"]
	preader := r.(utils.PasswordReader)
	owner := args[0]
	newPubKeyStr := args[1]
	newPriKeyStr := args[2]
	updateAccount, ok := mywallet.GetUnlockedAccount(owner)
	if !ok {
		fmt.Println(fmt.Sprintf("account: %s should be loaded or created first", owner))
		return
	}

	passphrase, err := utils.GetPassphrase(preader)
	if err != nil {
		fmt.Println(err)
		return
	}

	pubKey, err := prototype.PublicKeyFromWIF(newPubKeyStr)
	if err != nil {
		fmt.Println(err)
		return
	}

	accountUpdate_op := &prototype.AccountUpdateOperation{
		Owner:         &prototype.AccountName{Value: owner},
		Pubkey:        pubKey,
	}

	signTx, err := utils.GenerateSignedTxAndValidate(cmd, []interface{}{accountUpdate_op}, updateAccount)
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
			if resp.Invoice.Status == 200 {
				err = mywallet.Create(owner, passphrase, newPubKeyStr, newPriKeyStr)
				if err != nil {
					fmt.Println(err)
				}
			}
			fmt.Println(fmt.Sprintf("Result: %v", resp))
		}
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
