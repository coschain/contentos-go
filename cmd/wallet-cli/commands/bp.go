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

var bpUrlFlag string
var bpDescFlag string
var bpCreateAccountFee uint64
var bpBlockSize uint32
var bpVoteCancel bool

var BpCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use: "bp",
	}

	registerCmd := &cobra.Command{
		Use:     "register",
		Short:   "register as a new block-producer",
		Example: "bp register [bpname] [pubkey]",
		Args:    cobra.ExactArgs(2),
		Run:     registerBP,
	}

	registerCmd.Flags().StringVarP(&bpUrlFlag, "url", "u", "", `bp register alice --url "http://example.com"`)
	registerCmd.Flags().StringVarP(&bpDescFlag, "desc", "d", "", `bp register alice --desc "Hello World"`)
	registerCmd.Flags().Uint64VarP(&bpCreateAccountFee, "fee", "f", 1, `bp register alice --fee 1`)
	registerCmd.Flags().Uint32VarP(&bpBlockSize, "blocksize", "b", 1024*1024, `bp register alice --blocksize 1024`)

	unregisterCmd := &cobra.Command{
		Use:     "unregister",
		Short:   "unregister a block-producer",
		Example: "bp unregister [bpname]",
		Args:    cobra.ExactArgs(1),
		Run:     unRegisterBP,
	}

	voteCmd := &cobra.Command{
		Use:     "vote",
		Short:   "vote to a block-producer or unvote it",
		Example: "bp vote [voter] [bpname]",
		Args:    cobra.ExactArgs(2),
		Run:     voteBp,
	}

	voteCmd.Flags().BoolVarP(&bpVoteCancel, "cancel", "c", false, `bp vote alice bob --cancel`)

	cmd.AddCommand(registerCmd)
	cmd.AddCommand(unregisterCmd)
	cmd.AddCommand(voteCmd)

	return cmd
}

func registerBP(cmd *cobra.Command, args []string) {
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	mywallet := w.(wallet.Wallet)
	name := args[0]
	pubKeyStr := args[1]
	bpAccount, ok := mywallet.GetUnlockedAccount(name)
	if !ok {
		fmt.Println(fmt.Sprintf("account: %s should be loaded or created first", name))
		return
	}
	pubKey, err := prototype.PublicKeyFromWIF(pubKeyStr)
	if err != nil {
		fmt.Println(err)
		return
	}

	bpRegister_op := &prototype.BpRegisterOperation{
		Owner:           &prototype.AccountName{Value: name},
		Url:             bpUrlFlag,
		Desc:            bpDescFlag,
		BlockSigningKey: pubKey,
		Props: &prototype.ChainProperties{
			AccountCreationFee: prototype.NewCoin(bpCreateAccountFee),
			MaximumBlockSize:   bpBlockSize,
		},
	}

	signTx, err := utils.GenerateSignedTxAndValidate([]interface{}{bpRegister_op}, bpAccount)
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

func unRegisterBP(cmd *cobra.Command, args []string) {
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	mywallet := w.(wallet.Wallet)
	name := args[0]
	bpAccount, ok := mywallet.GetUnlockedAccount(name)
	if !ok {
		fmt.Println(fmt.Sprintf("account: %s should be loaded or created first", name))
		return
	}
	bpUnregister_op := &prototype.BpUnregisterOperation{
		Owner: &prototype.AccountName{Value: name},
	}

	signTx, err := utils.GenerateSignedTxAndValidate([]interface{}{bpUnregister_op}, bpAccount)
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

func voteBp(cmd *cobra.Command, args []string) {
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	mywallet := w.(wallet.Wallet)
	voter := args[0]
	voterAccount, ok := mywallet.GetUnlockedAccount(voter)
	bp := args[1]
	if !ok {
		fmt.Println(fmt.Sprintf("account: %s should be loaded or created first", voter))
		return
	}
	bpVote_op := &prototype.BpVoteOperation{
		Voter:   &prototype.AccountName{Value: voter},
		Witness: &prototype.AccountName{Value: bp},
		Cancel:  bpVoteCancel,
	}

	signTx, err := utils.GenerateSignedTxAndValidate([]interface{}{bpVote_op}, voterAccount)
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
