package commands

import (
	"bufio"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet-cli/commands/utils"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
	"github.com/coschain/contentos-go/rpc"
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
			keys := prototype.NewAuthorityFromPubKey(pubkey)

			acop := &prototype.AccountCreateOperation{
				Fee:            prototype.NewCoin(1),
				Creator:        &prototype.AccountName{Value: createrName},
				NewAccountName: &prototype.AccountName{Value: newAccountName},
				Owner:          keys,
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
				},
			}

			signTx, err = utils.GenerateSignedTxAndValidate2(client, []interface{}{bpRegister_op}, bpAccount)
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