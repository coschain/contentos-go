package commands

import (
	"bufio"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet-cli/commands/utils"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
	"github.com/coschain/contentos-go/rpc"
	"github.com/coschain/contentos-go/vm"
	"github.com/coschain/contentos-go/vm/context"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"context"
)

var BatchCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "conductBatch",
		Short:   "conduct batch commands",
		Example: "conduct [path]",
		Args:    cobra.ExactArgs(1),
		Run:     conductBatch,
	}
	return cmd
}

func conductBatch(cmd *cobra.Command, args []string) {
	var client grpcpb.ApiServiceClient
	var err error
	var signTx *prototype.SignedTransaction
	var path = args[0]
	file, err := os.Open(path)
	if err != nil {
		fmt.Println("can't open command batch file: ", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		cmdStr := scanner.Text()

		fmt.Println()
		fmt.Println("read string:",cmdStr)
		cmdArgs := strings.Split(cmdStr, " ")
		cmdType := cmdArgs[0]

		switch cmdType {
		case "switchport":
			checkpoint := cmdArgs[1]
			conn, err := rpc.Dial(checkpoint)
			if err != nil {
				fmt.Println("can't connect peer: ", checkpoint)
				return
			}
			client = grpcpb.NewApiServiceClient(conn)
			continue
		case "create":
			createrName := cmdArgs[1]
			createrPubKeyStr := cmdArgs[2]
			createrPriKeyStr := cmdArgs[3]
			newAccountName := cmdArgs[4]
			newAccountPubKeyStr := cmdArgs[5]

			creatorAccount := &wallet.PrivAccount{
				Account: wallet.Account{Name: createrName, PubKey: createrPubKeyStr},
				PrivKey: createrPriKeyStr,
			}

			pubkey, _ := prototype.PublicKeyFromWIF(newAccountPubKeyStr)

			acop := &prototype.AccountCreateOperation{
				Fee:            prototype.NewCoin(1),
				Creator:        &prototype.AccountName{Value: createrName},
				NewAccountName: &prototype.AccountName{Value: newAccountName},
				Owner:          pubkey,
			}

			signTx, err = utils.GenerateSignedTxAndValidate2(client, []interface{}{acop}, creatorAccount)
			if err != nil {
				fmt.Println(err)
				return
			}
		case "bpRegister":
			bpName := cmdArgs[1]
			bpPubKeyStr := cmdArgs[2]
			bpPriKeyStr := cmdArgs[3]

			bpAccount := &wallet.PrivAccount{
				Account: wallet.Account{Name: bpName, PubKey: bpPubKeyStr},
				PrivKey: bpPriKeyStr,
			}

			pubkey, _ := prototype.PublicKeyFromWIF(bpPubKeyStr)

			bpRegister_op := &prototype.BpRegisterOperation{
				Owner:           &prototype.AccountName{Value: bpName},
				Url:             bpUrlFlag,
				Desc:            bpDescFlag,
				BlockSigningKey: pubkey,
				Props: &prototype.ChainProperties{
					AccountCreationFee: prototype.NewCoin(bpCreateAccountFee),
					MaximumBlockSize:   bpBlockSize,
					StaminaFree:        constants.DefaultStaminaFree,
					TpsExpected:        constants.DefaultTPSExpected,
				},
			}

			signTx, err = utils.GenerateSignedTxAndValidate2(client, []interface{}{bpRegister_op}, bpAccount)
			if err != nil {
				fmt.Println(err)
				return
			}
		case "bpVote":
			voterName := cmdArgs[1]
			voterPubKeyStr := cmdArgs[2]
			voterPriKeyStr := cmdArgs[3]
			bpName := cmdArgs[4]

			voterAccount := &wallet.PrivAccount{
				Account: wallet.Account{Name: voterName, PubKey: voterPubKeyStr},
				PrivKey: voterPriKeyStr,
			}

			bpVote_op := &prototype.BpVoteOperation{
				Voter:   &prototype.AccountName{Value: voterName},
				Witness: &prototype.AccountName{Value: bpName},
				Cancel:  false,
			}

			signTx, err = utils.GenerateSignedTxAndValidate2(client, []interface{}{bpVote_op}, voterAccount)
			if err != nil {
				fmt.Println(err)
				return
			}
		case "transfer":
			fromAccountName := cmdArgs[1]
			fromAccountPubKeyStr := cmdArgs[2]
			fromAccountPriKeyStr := cmdArgs[3]
			toAccountName := cmdArgs[4]
			amount, err := strconv.ParseUint(cmdArgs[5], 10, 64)
			if err != nil {
				panic(err)
			}

			fromAccount := &wallet.PrivAccount{
				Account: wallet.Account{Name: fromAccountName, PubKey: fromAccountPubKeyStr},
				PrivKey: fromAccountPriKeyStr,
			}

			transfer_op := &prototype.TransferOperation{
				From:   &prototype.AccountName{Value: fromAccountName},
				To:     &prototype.AccountName{Value: toAccountName},
				Amount: prototype.NewCoin(amount),
				Memo:   "",
			}

			signTx, err = utils.GenerateSignedTxAndValidate2(client, []interface{}{transfer_op}, fromAccount)
			if err != nil {
				fmt.Println(err)
				return
			}
		case "transferToVest":
			fromAccountName := cmdArgs[1]
			fromAccountPubKeyStr := cmdArgs[2]
			fromAccountPriKeyStr := cmdArgs[3]
			toAccountName := cmdArgs[4]
			amount, err := strconv.ParseUint(cmdArgs[5], 10, 64)
			if err != nil {
				panic(err)
			}

			fromAccount := &wallet.PrivAccount{
				Account: wallet.Account{Name: fromAccountName, PubKey: fromAccountPubKeyStr},
				PrivKey: fromAccountPriKeyStr,
			}

			transferv_op := &prototype.TransferToVestingOperation{
				From:   &prototype.AccountName{Value: fromAccountName},
				To:     &prototype.AccountName{Value: toAccountName},
				Amount: prototype.NewCoin(uint64(amount)),
			}

			signTx, err = utils.GenerateSignedTxAndValidate2(client, []interface{}{transferv_op}, fromAccount)
			if err != nil {
				fmt.Println(err)
				return
			}
		case "deploy":
			deployerName := cmdArgs[1]
			deployerPubKeyStr := cmdArgs[2]
			deployerPriKeyStr := cmdArgs[3]
			contractName := cmdArgs[4]
			wasmPath:= cmdArgs[5]
			abiPath:= cmdArgs[6]

			code, _ := ioutil.ReadFile(wasmPath)
			abi, _ := ioutil.ReadFile(abiPath)

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

			deployerAccount := &wallet.PrivAccount{
				Account: wallet.Account{Name: deployerName, PubKey: deployerPubKeyStr},
				PrivKey: deployerPriKeyStr,
			}

			contractDeployOp := &prototype.ContractDeployOperation{
				Owner:    &prototype.AccountName{Value: deployerName},
				Contract: contractName,
				Abi:      compressedAbi,
				Code:     compressedCode,
				Upgradeable: false,
			}

			signTx, err = utils.GenerateSignedTxAndValidate2(client, []interface{}{contractDeployOp}, deployerAccount)
			if err != nil {
				fmt.Println(err)
				return
			}
		case "stake":
			stakerName := cmdArgs[1]
			stakerPubKeyStr := cmdArgs[2]
			stakerPriKeyStr := cmdArgs[3]

			amount, err := strconv.ParseInt(cmdArgs[4], 10, 64)
			if err != nil {
				fmt.Println(err)
				return
			}

			stakeAccount := &wallet.PrivAccount{
				Account: wallet.Account{Name: stakerName, PubKey: stakerPubKeyStr},
				PrivKey: stakerPriKeyStr,
			}

			stakeOp := &prototype.StakeOperation{
				Account:   &prototype.AccountName{Value: stakerName},
				Amount:    prototype.NewCoin(uint64(amount)),
			}

			signTx, err = utils.GenerateSignedTxAndValidate2(client, []interface{}{stakeOp}, stakeAccount)
			if err != nil {
				fmt.Println(err)
				return
			}
		default:
			fmt.Println("unknown command")
			continue
		}

		req := &grpcpb.BroadcastTrxRequest{Transaction: signTx}
		resp, err := client.BroadcastTrx(context.Background(), req)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(fmt.Sprintf("Result: %v", resp))
		}
	}
}