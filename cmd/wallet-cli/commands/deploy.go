package commands

import (
	"context"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet-cli/commands/utils"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
	"github.com/coschain/contentos-go/vm"
	"github.com/coschain/contentos-go/vm/context"
	"io/ioutil"
)

var DeployCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deploy",
		Short:   "deploy a new contract",
		Example: "deploy [author] [contract_name] [local_wasm_path] [local_abi_path] [upgradeable]",
		Args:    cobra.ExactArgs(5),
		Run:     deploy,
	}
	return cmd
}

func deploy(cmd *cobra.Command, args []string) {
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	mywallet := w.(wallet.Wallet)
	author := args[0]
	acc, ok := mywallet.GetUnlockedAccount(author)
	if !ok {
		fmt.Println(fmt.Sprintf("author: %s should be loaded or created first", author))
		return
	}
	cname := args[1]
	path := args[2]
	pathAbi := args[3]

	upgradeable := false
	if args[4] == "true"{
		upgradeable = true
	}

	code, _ := ioutil.ReadFile(path)
	abi, _ := ioutil.ReadFile(pathAbi)

	// code and abi compression
	var (
		compressedCode, compressedAbi []byte
		err error
	)
	if compressedCode, err = common.Compress(code); err != nil {
		fmt.Println(fmt.Sprintf("code compression failed: %s", err.Error()))
		return
	}
	if compressedAbi, err = common.Compress(abi); err != nil {
		fmt.Println(fmt.Sprintf("abi compression failed: %s", err.Error()))
		return
	}

	ctx := vmcontext.Context{Code: code}
	cosVM := vm.NewCosVM(&ctx, nil, nil, nil)
	err = cosVM.Validate()
	if err != nil {
		fmt.Println("Validate local code error:", err)
		return
	}
	contractDeployOp := &prototype.ContractDeployOperation{
		Owner:    &prototype.AccountName{Value: author},
		Contract: cname,
		Abi:      compressedAbi,
		Code:     compressedCode,
		Upgradeable:upgradeable,
	}
	signTx, err := utils.GenerateSignedTxAndValidate2(client, []interface{}{contractDeployOp}, acc)
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
