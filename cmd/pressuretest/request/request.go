package request

import (
	"context"
	"fmt"
	"github.com/coschain/contentos-go/cmd/wallet-cli/commands/utils"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
	"math/rand"
	"strings"
	"time"
)

var nameLib = "abcdefghijklmnopqrstuvwxyz01234567890"

func createAccount(mywallet *wallet.BaseWallet, rpcClient grpcpb.ApiServiceClient, creatorAccount *wallet.PrivAccount, newAccountName string) {

	if creatorAccount == nil {
		GlobalAccountLIst.RLock()
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		idx := r.Intn( len(GlobalAccountLIst.arr) )
		creatorAccount = GlobalAccountLIst.arr[idx]
		GlobalAccountLIst.RUnlock()
	}

	// if not specify a name, generate a random one
	if newAccountName == "" {
		for {
			for i:=0;i<15;i++{
				r := rand.New(rand.NewSource(time.Now().UnixNano()))
				idx := r.Intn(len(nameLib))
				newAccountName += string(nameLib[idx])
			}
			if creatorAccount.Name != newAccountName {
				break
			}
		}
	}

	pubKeyStr, privKeyStr, err := mywallet.GenerateNewKey()
	if err != nil {
		fmt.Println(err)
		return
	}
	pubkey, _ := prototype.PublicKeyFromWIF(pubKeyStr)
	keys := prototype.NewAuthorityFromPubKey(pubkey)

	acop := &prototype.AccountCreateOperation{
		Fee:            prototype.NewCoin(1),
		Creator:        &prototype.AccountName{Value: creatorAccount.Name},
		NewAccountName: &prototype.AccountName{Value: newAccountName},
		Owner:          keys,
	}
	signTx, err := utils.GenerateSignedTxAndValidate2(rpcClient, []interface{}{acop}, creatorAccount)
	if err != nil {
		fmt.Println(err)
		return
	}
	req := &grpcpb.BroadcastTrxRequest{Transaction: signTx}
	resp, err := rpcClient.BroadcastTrx(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		if resp.Invoice.Status == 200 {
			GlobalAccountLIst.Lock()
			obj := &wallet.PrivAccount{
				Account: wallet.Account{Name: newAccountName, PubKey: pubKeyStr},
				PrivKey: privKeyStr,
			}
			GlobalAccountLIst.arr = append(GlobalAccountLIst.arr, obj)
			GlobalAccountLIst.Unlock()
		}

		if strings.Contains(resp.Invoice.ErrorInfo, "Insufficient") {
			transfer(rpcClient, GlobalAccountLIst.arr[0], creatorAccount, 10)
			createAccount(mywallet, rpcClient, creatorAccount, newAccountName)
			return
		}
		fmt.Println(fmt.Sprintf("Result: %v", resp))
	}
}

func transfer(rpcClient grpcpb.ApiServiceClient, fromAccount, toAccount  *wallet.PrivAccount, amount int) {
	if fromAccount == nil {
		GlobalAccountLIst.RLock()
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		idx := r.Intn( len(GlobalAccountLIst.arr) )
		fromAccount = GlobalAccountLIst.arr[idx]
		GlobalAccountLIst.RUnlock()
	}

	if toAccount == nil {
		for {
			GlobalAccountLIst.RLock()
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			idx := r.Intn( len(GlobalAccountLIst.arr) )
			toAccount = GlobalAccountLIst.arr[idx]
			GlobalAccountLIst.RUnlock()
			if fromAccount != toAccount {
				break
			}
		}
	}

	if amount == 0 {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		amount = 1 + r.Intn(10)
	}

	transfer_op := &prototype.TransferOperation{
		From:   &prototype.AccountName{Value: fromAccount.Name},
		To:     &prototype.AccountName{Value: toAccount.Name},
		Amount: prototype.NewCoin(uint64(amount)),
		Memo:   "",
	}
	signTx, err := utils.GenerateSignedTxAndValidate2(rpcClient, []interface{}{transfer_op}, fromAccount)
	if err != nil {
		fmt.Println(err)
		return
	}
	req := &grpcpb.BroadcastTrxRequest{Transaction: signTx}
	resp, err := rpcClient.BroadcastTrx(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		if strings.Contains(resp.Invoice.ErrorInfo, "Insufficient") {
			transfer(rpcClient, GlobalAccountLIst.arr[0], fromAccount, 10)
			transfer(rpcClient, fromAccount, toAccount, 0)
			return
		}

		fmt.Println(fmt.Sprintf("Result: %v", resp))
	}
}

func postArticle(rpcClient grpcpb.ApiServiceClient, authorAccount *wallet.PrivAccount) {

	if authorAccount == nil {
		GlobalAccountLIst.RLock()
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		idx := r.Intn( len(GlobalAccountLIst.arr) )
		authorAccount = GlobalAccountLIst.arr[idx]
		GlobalAccountLIst.RUnlock()
	}

	var tag = ""
	var title = ""
	var content = ""
	beneficiaries := []*prototype.BeneficiaryRouteType{}
	for i:=0;i<10;i++ {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		idx := r.Intn(len(nameLib))
		tag += string(nameLib[idx])
	}
	for i:=0;i<20;i++ {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		idx := r.Intn(len(nameLib))
		title += string(nameLib[idx])
	}
	for i:=0;i<1024;i++ {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		idx := r.Intn(len(nameLib))
		content += string(nameLib[idx])
	}

	uuid := utils.GenerateUUID( authorAccount.Name + title )
	post_op := &prototype.PostOperation{
		Uuid:          uuid,
		Owner:         &prototype.AccountName{Value: authorAccount.Name},
		Title:         title,
		Content:       content,
		Tags:          []string{tag},
		Beneficiaries: beneficiaries,
	}
	signTx, err := utils.GenerateSignedTxAndValidate2(rpcClient, []interface{}{post_op}, authorAccount)
	if err != nil {
		fmt.Println(err)
		return
	}
	req := &grpcpb.BroadcastTrxRequest{Transaction: signTx}
	resp, err := rpcClient.BroadcastTrx(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(fmt.Sprintf("Result: %v", resp))
	}
}

func follow(rpcClient grpcpb.ApiServiceClient, followerAccount, followingAccount *wallet.PrivAccount) {
	if followerAccount == nil {
		GlobalAccountLIst.RLock()
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		idx := r.Intn( len(GlobalAccountLIst.arr) )
		followerAccount = GlobalAccountLIst.arr[idx]
		GlobalAccountLIst.RUnlock()
	}

	if followingAccount == nil {
		for {
			GlobalAccountLIst.RLock()
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			idx := r.Intn( len(GlobalAccountLIst.arr) )
			followingAccount = GlobalAccountLIst.arr[idx]
			GlobalAccountLIst.RUnlock()
			if followerAccount != followingAccount {
				break
			}
		}
	}

	follow_op := &prototype.FollowOperation{
		Account:  &prototype.AccountName{Value: followerAccount.Name},
		FAccount: &prototype.AccountName{Value: followingAccount.Name},
		Cancel:   false,
	}

	signTx, err := utils.GenerateSignedTxAndValidate2(rpcClient, []interface{}{follow_op}, followerAccount)
	if err != nil {
		fmt.Println(err)
		return
	}
	req := &grpcpb.BroadcastTrxRequest{Transaction: signTx}
	resp, err := rpcClient.BroadcastTrx(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(fmt.Sprintf("Result: %v", resp))
	}
}