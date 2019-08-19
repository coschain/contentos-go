package commands

import (
	"context"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet-cli/commands/utils"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
	"math/rand"
	"strconv"
	"sync"
)

var RTransferCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "rtran",
		Short:   "rtran <thread-count> <acc_name_prefix> <acc_count>",
		Example: "rtran 2 user 1000",
		Args:    cobra.ExactArgs(3),
		Run:     rTransfer,
	}
	return cmd
}

func rTransfer(cmd *cobra.Command, args []string) {
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	//w := cmd.Context["wallet"]
	//mywallet := w.(wallet.Wallet)
	chainId := cmd.Context["chain_id"].(prototype.ChainId)

	tCount, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Println(err)
		return
	}
	prefix := args[1]
	accCount, err := strconv.Atoi(args[2])
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("getting private keys...")
	keys := make([]*prototype.PrivateKeyType, accCount)
	for i := range keys {
		accName := fmt.Sprintf("%s%d", prefix, i)
		keys[i], _ = prototype.FixBytesToPrivateKey([]byte(accName))
	}

	wg := &sync.WaitGroup{}
	for i := 0; i < tCount; i++  {
		wg.Add(1)
		go func(){
			for index:=0; index < 1000; index ++ {
				a, b := 0, 0
				for a == b {
					a = rand.Intn(accCount)
					b = rand.Intn(accCount)
				}
				from := fmt.Sprintf("%s%d", prefix, a)
				to := fmt.Sprintf("%s%d", prefix, b)

				transferOp := &prototype.TransferOperation{
					From:   &prototype.AccountName{Value: from},
					To:     &prototype.AccountName{Value: to},
					Amount: prototype.NewCoin(uint64(1)),
					Memo:   randStr(8),
				}
				signTx, err := utils.GenerateSignedTxAndValidate3(client, []interface{}{transferOp}, keys[a], chainId)
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

				if res.Status != prototype.StatusSuccess {
					fmt.Println(res)
					break
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
}
