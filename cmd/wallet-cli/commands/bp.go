package commands

import (
	"context"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
)

var bpUrlFlag string
var bpDescFlag string
var bpCreateAccountFee int64
var bpBlockSize uint32

var BpCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use: "bp",
	}

	registerCmd := &cobra.Command{
		Use:   "register",
		Short: "register as a new block-producer",
		Args:  cobra.ExactArgs(1),
		Run:   registerBP,
	}

	registerCmd.Flags().StringVarP(&bpUrlFlag, "url", "u", "", `import --url "http://example.com"`)
	registerCmd.Flags().StringVarP(&bpDescFlag, "desc", "d", "", `import --desc "Hello World"`)
	registerCmd.Flags().Int64VarP(&bpCreateAccountFee, "fee", "", 1, `import --fee 1`)
	registerCmd.Flags().Uint32VarP(&bpBlockSize, "blocksize", "", 1024*1024, `import --blocksize 1024`)

	cmd.AddCommand(registerCmd)

	return cmd
}

func registerBP(cmd *cobra.Command, args []string) {
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	mywallet := w.(*wallet.BaseWallet)
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

	bpregister_op := &prototype.BpRegisterOperation{
		Owner:           &prototype.AccountName{Value: name},
		Url:             bpUrlFlag,
		Desc:            bpDescFlag,
		BlockSigningKey: pubKey,
		Props: &prototype.ChainProperties{
			AccountCreationFee: &prototype.Coin{Amount: &prototype.Safe64{Value: bpCreateAccountFee}},
			MaximumBlockSize:   bpBlockSize,
		},
	}

	signTx, err := GenerateSignedTx([]interface{}{bpregister_op}, bpAccount)
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
