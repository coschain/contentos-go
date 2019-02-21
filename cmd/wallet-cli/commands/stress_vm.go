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
	"sync"
	"time"
)

var StressVMCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "stressvm",
		Short:   "stressvm [count] [caller] [owner] [contract_name]",
		Long:    "stress vm call",
		Example: "stress 2 initminer initminer print",
		Args:    cobra.MinimumNArgs(4),
		Run:     stressVM,
	}
	return cmd
}

func stressVM(cmd *cobra.Command, args []string) {
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	mywallet := w.(wallet.Wallet)
	num, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Println(err)
		return
	}
	caller := args[1]
	owner := args[2]
	cname := args[3]
	acc, ok := mywallet.GetUnlockedAccount(caller)
	if !ok {
		fmt.Println(fmt.Sprintf("caller: %s should be loaded or created first", caller))
		return
	}
	wg := &sync.WaitGroup{}
	for i := 0; i < num; i++ {
		tid := i
		wg.Add(1)
		go func() {
			s := time.Now()
			defer func() {
				fmt.Println("stress cost: ", time.Now().Sub(s), ", thread-number: ", tid)
			}()
			for index := 0; index < 1000; index++ {
				contractApplyOp := &prototype.ContractApplyOperation{
					Caller:   &prototype.AccountName{Value: caller},
					Owner:    &prototype.AccountName{Value: owner},
					Contract: cname,
					Params:   strconv.Itoa(tid) + "*t*" + strconv.Itoa(index),
				}
				signTx, err := utils.GenerateSignedTxAndValidate2(client, []interface{}{contractApplyOp}, acc)
				if err != nil {
					fmt.Println(err)
					return
				}

				req := &grpcpb.BroadcastTrxRequest{Transaction: signTx}
				res, err := client.BroadcastTrx(context.Background(), req)

				if err != nil {
					fmt.Println(err)
					break
				}

				if !res.Invoice.IsSuccess() {
					fmt.Println(res)
					break
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
}
