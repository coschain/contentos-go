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

var StressCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "stress",
		Short:   "stress thread-count initminer alicex 1",
		Long:    "stress thread-count transfer cos from one account to another account",
		Example: "stress 2 initminer alicex 1",
		Args:    cobra.MinimumNArgs(4),
		Run:     stress,
	}
	return cmd
}

func stress(cmd *cobra.Command, args []string) {
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	mywallet := w.(wallet.Wallet)
	from := args[1]
	to := args[2]
	amount, err := utils.ParseCos(args[3])
	if err != nil {
		fmt.Println(err)
		return
	}
	tCount, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Println(err)
		return
	}
	memo := ""
	if len(args) > 3 {
		memo = args[4]
	}
	fromAccount, ok := mywallet.GetUnlockedAccount(from)
	if !ok {
		fmt.Println(fmt.Sprintf("account: %s should be loaded or created first", from))
		return
	}
	wg := &sync.WaitGroup{}
	for i := 0; i < tCount; i++  {
		tid := i
		wg.Add(1)
		go func(){
			s := time.Now()
			defer func() {
				fmt.Println("stress cost: ", time.Now().Sub(s), ", thread-number: ", tid)
			}()
			for index:=0; index < 1000; index ++ {
				transferOp := &prototype.TransferOperation{
					From:   &prototype.AccountName{Value: from},
					To:     &prototype.AccountName{Value: to},
					Amount: prototype.NewCoin(amount),
					Memo:   strconv.Itoa(tid) + "*t*" + memo + strconv.Itoa(index),
				}
				signTx, err := utils.GenerateSignedTxAndValidate(cmd, []interface{}{transferOp}, fromAccount)
				if err != nil {
					fmt.Println(err)
					return
				}

				req := &grpcpb.BroadcastTrxRequest{Transaction: signTx}
				req.OnlyDeliver = true
				res, err := client.BroadcastTrx(context.Background(), req)

				if err != nil {
					fmt.Println(err)
					break
				}

				if res.Status != prototype.StatusSuccess && res.Status != prototype.StatusDeductStamina{
					fmt.Println(res)
					break
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
}
