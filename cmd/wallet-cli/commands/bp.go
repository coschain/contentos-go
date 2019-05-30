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
)

var bpUrlFlag string
var bpDescFlag string
var bpCreateAccountFee uint64
var bpBlockSize uint32
var bpVoteCancel bool
var proposedStaminaFree uint64
var bpUpdateStaminaFree uint64
var tpsExpected uint64
var bpUpdateTpsExpected uint64
var bpUpdateCreateAccountFee uint64

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
	registerCmd.Flags().Uint64VarP(&bpCreateAccountFee, "fee", "f", constants.DefaultAccountCreateFee, `bp register alice --fee 1`)
	registerCmd.Flags().Uint32VarP(&bpBlockSize, "blocksize", "b", 1024*1024, `bp register alice --blocksize 1024`)
	registerCmd.Flags().Uint64VarP(&proposedStaminaFree, "stamina_free", "s", constants.DefaultStaminaFree, `bp register alice --stamina_free 1`)
	registerCmd.Flags().Uint64VarP(&tpsExpected, "tps", "t", constants.DefaultTPSExpected, `bp register alice --tps 1`)

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

	updateCmd := &cobra.Command{
		Use:     "update",
		Short:   "update block-producer state",
		Example: "bp update [bpname] --xxx xx",
		Args:    cobra.ExactArgs(1),
		Run:     updateBp,
	}

	updateCmd.Flags().Uint64VarP(&bpUpdateStaminaFree, "stamina_free", "s", constants.DefaultStaminaFree, `bp update alice --stamina_free 1`)
	updateCmd.Flags().Uint64VarP(&bpUpdateTpsExpected, "tps", "t", constants.DefaultTPSExpected, `bp update alice --tps 1`)
	updateCmd.Flags().Uint64VarP(&bpUpdateCreateAccountFee, "fee", "f", constants.DefaultAccountCreateFee, `bp update alice --fee 1`)

	cmd.AddCommand(registerCmd)
	cmd.AddCommand(unregisterCmd)
	cmd.AddCommand(voteCmd)
	cmd.AddCommand(updateCmd)

	return cmd
}

func registerBP(cmd *cobra.Command, args []string) {
	defer func() {
		// reset to default value
		// it's hard to assign default value from cobra.command
		// so I have to do it manually
		bpCreateAccountFee = constants.DefaultAccountCreateFee
		bpBlockSize = 1024 * 1024
		bpUrlFlag = ""
		bpDescFlag = ""
		proposedStaminaFree = constants.DefaultStaminaFree
		tpsExpected = constants.DefaultTPSExpected
	}()
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
			StaminaFree:        proposedStaminaFree,
			TpsExpected:        tpsExpected,
		},
	}

	signTx, err := utils.GenerateSignedTxAndValidate2(client, []interface{}{bpRegister_op}, bpAccount)
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

	signTx, err := utils.GenerateSignedTxAndValidate2(client, []interface{}{bpUnregister_op}, bpAccount)
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
	defer func() {
		bpVoteCancel = false
	}()
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

	signTx, err := utils.GenerateSignedTxAndValidate2(client, []interface{}{bpVote_op}, voterAccount)
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

func updateBp(cmd *cobra.Command, args []string) {
	defer func() {
		bpUpdateStaminaFree      = constants.DefaultStaminaFree
		bpUpdateTpsExpected      = constants.DefaultTPSExpected
		bpUpdateCreateAccountFee = constants.DefaultAccountCreateFee
	}()
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
	bpUpdate_op := &prototype.BpUpdateOperation{
		Owner:                 &prototype.AccountName{Value: name},
		ProposedStaminaFree:   bpUpdateStaminaFree,
		TpsExpected:           bpUpdateTpsExpected,
		AccountCreationFee:    prototype.NewCoin(bpUpdateCreateAccountFee),
	}

	signTx, err := utils.GenerateSignedTxAndValidate2(client, []interface{}{bpUpdate_op}, bpAccount)
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
