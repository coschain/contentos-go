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
)

var CreateFundAccountCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "createfund",
		Short:   "createfund <creator> <acc_name_prefix> <acc_count> <fund_balance>",
		Example: "createfund initminer user 1000 10",
		Args:    cobra.ExactArgs(4),
		Run:     createFundAccount,
	}
	return cmd
}

func createFundAccount(cmd *cobra.Command, args []string) {
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	mywallet := w.(wallet.Wallet)
	creator := args[0]
	prefix := args[1]
	accCount, _ := strconv.Atoi(args[2])
	fundBalance, err := utils.ParseCos(args[3])
	if err != nil {
		fmt.Println(err)
		return
	}
	tCount := 100
	tJobs := accCount / tCount
	creatorAccount, ok := mywallet.GetUnlockedAccount(creator)
	if !ok {
		fmt.Println(fmt.Sprintf("account: %s should be loaded or created first", creator))
		return
	}
	isWait := true

	wg := &sync.WaitGroup{}
	for i := 0; i < tCount; i++  {
		//tid := i
		wg.Add(1)
		d := tJobs
		if i == tCount - 1 {
			d = accCount - tJobs * i
		}
		go func(start, end int){
			for index:=start; index < end; index ++ {
				accName := fmt.Sprintf("%s%d", prefix, index)
				privKey, err := prototype.GenerateNewKeyFromBytes([]byte(accName))
				if err != nil {
					fmt.Println(err)
					break
				}
				pubKey, err := privKey.PubKey()
				if err != nil {
					fmt.Println(err)
					break
				}
				acop := &prototype.AccountCreateOperation{
					Fee:            prototype.NewCoin(uint64(fundBalance)),
					Creator:        &prototype.AccountName{Value: creator},
					NewAccountName: &prototype.AccountName{Value: accName},
					Owner:          pubKey,
				}
				fop := &prototype.TransferOperation{
					From: &prototype.AccountName{Value: creator},
					To: &prototype.AccountName{Value: accName},
					Amount: prototype.NewCoin(uint64(fundBalance)),
					Memo: randStr(8),
				}
				signTx, err := utils.GenerateSignedTxAndValidate(cmd, []interface{}{acop, fop}, creatorAccount)
				if err != nil {
					fmt.Println(err)
					return
				}

				req := &grpcpb.BroadcastTrxRequest{Transaction: signTx}
				if isWait {
					res, err := client.BroadcastTrx(context.Background(), req)
					if err != nil {
						fmt.Println(err)
						break
					}

					if !res.Invoice.IsSuccess() {
						fmt.Println(res)
						break
					}
				}else {
					go func() {
						res, err := client.BroadcastTrx(context.Background(), req)
						if err != nil {
							fmt.Println(err)
							return
						}

						if !res.Invoice.IsSuccess() {
							fmt.Println(res)
							return
						}
					}()
				}

			}
			wg.Done()
		}(tJobs * i, tJobs * i + d)
	}
	wg.Wait()
}
