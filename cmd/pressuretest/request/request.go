package request

import (
	"context"
	"fmt"
	"github.com/coschain/contentos-go/cmd/wallet-cli/commands/utils"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
	"math/rand"
	"strings"
	"time"
	"errors"
)

var nameLib = "abcdefghijklmnopqrstuvwxyz01234567890"

func stake(rpcClient grpcpb.ApiServiceClient, from *wallet.PrivAccount,to *wallet.PrivAccount, amount uint64) {
	stkop := &prototype.StakeOperation{
		From:        &prototype.AccountName{Value: from.Name},
		To:        &prototype.AccountName{Value: to.Name},
		Amount:            &prototype.Coin{Value: amount},
	}

	signTx, err := utils.GenerateSignedTxAndValidate2(rpcClient, []interface{}{stkop}, from, ChainId)
	if err != nil {
		fmt.Println(err)
		return
	}

	//fmt.Println("Request command: ", fmt.Sprintf("create %s %s", creatorAccount.Name, newAccountName) )

	req := &grpcpb.BroadcastTrxRequest{Transaction: signTx}
	resp, err := rpcClient.BroadcastTrx(context.Background(), req)
	if err != nil {
		fmt.Println("stake error:",err)
	} else {
		fmt.Println("Request command: ",
			fmt.Sprintf("stake from %s to %s", from.Name,to.Name),
			" ",
			fmt.Sprintf("Result: %v", resp))
	}
}

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

	acop := &prototype.AccountCreateOperation{
		Fee:            prototype.NewCoin(constants.DefaultAccountCreateFee),
		Creator:        &prototype.AccountName{Value: creatorAccount.Name},
		NewAccountName: &prototype.AccountName{Value: newAccountName},
		PubKey:          pubkey,
	}
	signTx, err := utils.GenerateSignedTxAndValidate2(rpcClient, []interface{}{acop}, creatorAccount, ChainId)
	if err != nil {
		fmt.Println(err)
		return
	}

	//fmt.Println("Request command: ", fmt.Sprintf("create %s %s", creatorAccount.Name, newAccountName) )

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
			err := transfer(rpcClient, GlobalAccountLIst.arr[0], creatorAccount, 5)
			if err != nil {
				fmt.Println(err)
				return
			}
			err = vest(rpcClient, GlobalAccountLIst.arr[0], creatorAccount, 5)
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(fmt.Sprintf("====== createaccount from:%v to:%v amount:%v",GlobalAccountLIst.arr[0].Name,creatorAccount.Name,5))
			createAccount(mywallet, rpcClient, creatorAccount, newAccountName)
			return
		}
		if strings.Contains(resp.Invoice.ErrorInfo,"net resource not enough") {
			stake(rpcClient,creatorAccount,creatorAccount,1)
		}
		fmt.Println("Request command: ",
			fmt.Sprintf("create %s %s", creatorAccount.Name, newAccountName),
			" ",
			fmt.Sprintf("Result: %v", resp))
	}
	// give new account 1 coin and let him stake
	toAccount := &wallet.PrivAccount{}
	toAccount.Name = newAccountName
	stake(rpcClient,creatorAccount,toAccount,1)
}

func transfer(rpcClient grpcpb.ApiServiceClient, fromAccount, toAccount  *wallet.PrivAccount, amount int) error {
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
		amount = 1 + r.Intn(5)
	}

	transfer_op := &prototype.TransferOperation{
		From:   &prototype.AccountName{Value: fromAccount.Name},
		To:     &prototype.AccountName{Value: toAccount.Name},
		Amount: prototype.NewCoin(uint64(amount)),
		Memo:   "",
	}
	signTx, err := utils.GenerateSignedTxAndValidate2(rpcClient, []interface{}{transfer_op}, fromAccount, ChainId)
	if err != nil {
		fmt.Println(err)
		return err
	}

	//fmt.Println("Request command: ", fmt.Sprintf("transfer %s %s %d", fromAccount.Name, toAccount.Name, amount) )

	req := &grpcpb.BroadcastTrxRequest{Transaction: signTx}
	resp, err := rpcClient.BroadcastTrx(context.Background(), req)
	if err != nil {
		fmt.Println(err)
		return err
	} else {
		if strings.Contains(resp.Invoice.ErrorInfo, "Insufficient") {
			if fromAccount == GlobalAccountLIst.arr[0] {
				return errors.New("initminer has no money left")
			}
			err := transfer(rpcClient, GlobalAccountLIst.arr[0], fromAccount, 5)
			if err != nil {
				fmt.Println(err)
				return err
			}
			err = vest(rpcClient, GlobalAccountLIst.arr[0], fromAccount, 5)
			if err != nil {
				fmt.Println(err)
				return err
			}
			fmt.Println(fmt.Sprintf("====== transfer from:%v to:%v amount:%v",GlobalAccountLIst.arr[0].Name,fromAccount.Name,5))
			transfer(rpcClient, fromAccount, toAccount, amount)
			return nil
		}

		if strings.Contains(resp.Invoice.ErrorInfo,"net resource not enough") {
			stake(rpcClient,fromAccount,fromAccount,1)
		}

		fmt.Println("Request command: ",
			fmt.Sprintf("transfer %s %s %d", fromAccount.Name, toAccount.Name, amount),
			" ",
			fmt.Sprintf("Result: %v", resp))
	}
	return nil
}

func vest(rpcClient grpcpb.ApiServiceClient, fromAccount, toAccount  *wallet.PrivAccount, amount int) error {
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

	transfer_op := &prototype.TransferToVestingOperation{
		From:   &prototype.AccountName{Value: fromAccount.Name},
		To:     &prototype.AccountName{Value: toAccount.Name},
		Amount: prototype.NewCoin(uint64(amount)),
	}
	signTx, err := utils.GenerateSignedTxAndValidate2(rpcClient, []interface{}{transfer_op}, fromAccount, ChainId)
	if err != nil {
		fmt.Println(err)
		return err
	}

	//fmt.Println("Request command: ", fmt.Sprintf("transfer vest %s %s %d", fromAccount.Name, toAccount.Name, amount) )

	req := &grpcpb.BroadcastTrxRequest{Transaction: signTx}
	resp, err := rpcClient.BroadcastTrx(context.Background(), req)
	if err != nil {
		fmt.Println(err)
		return err
	} else {
		if strings.Contains(resp.Invoice.ErrorInfo, "Insufficient") {
			if fromAccount == GlobalAccountLIst.arr[0] {
				return errors.New("initminer has no money left")
			}
			err := vest(rpcClient, GlobalAccountLIst.arr[0], fromAccount, 10)
			if err != nil {
				fmt.Println(err)
				return err
			}
			fmt.Println(fmt.Sprintf("====== vest from:%v to:%v amount:%v",GlobalAccountLIst.arr[0].Name,fromAccount.Name,5))
			vest(rpcClient, fromAccount, toAccount, amount)
			return nil
		}

		if strings.Contains(resp.Invoice.ErrorInfo,"net resource not enough") {
			stake(rpcClient,fromAccount,fromAccount,1)
		}

		fmt.Println("Request command: ",
			fmt.Sprintf("transfer vest %s %s %d", fromAccount.Name, toAccount.Name, amount),
			" ",
			fmt.Sprintf("Result: %v", resp))
	}
	return nil
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
	signTx, err := utils.GenerateSignedTxAndValidate2(rpcClient, []interface{}{post_op}, authorAccount, ChainId)
	if err != nil {
		fmt.Println(err)
		return
	}

	//fmt.Println("Request command: ", fmt.Sprintf("%s post an article", authorAccount.Name) )

	req := &grpcpb.BroadcastTrxRequest{Transaction: signTx}
	resp, err := rpcClient.BroadcastTrx(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		if resp.Invoice.Status == 200 {
			PostIdList.Lock()
			PostIdList.arr = append(PostIdList.arr, uuid)
			PostIdList.Unlock()
		}

		if strings.Contains(resp.Invoice.ErrorInfo, "Insufficient") {
			err := transfer(rpcClient, GlobalAccountLIst.arr[0], authorAccount, 5)
			if err != nil {
				fmt.Println(err)
				return
			}
			err = vest(rpcClient, GlobalAccountLIst.arr[0], authorAccount, 5)
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(fmt.Sprintf("====== post transfer from:%v to:%v amount:%v",GlobalAccountLIst.arr[0].Name,authorAccount.Name,5))
			postArticle(rpcClient, authorAccount)
			return
		}
		if strings.Contains(resp.Invoice.ErrorInfo,"net resource not enough") {
			stake(rpcClient,authorAccount,authorAccount,1)
		}

		fmt.Println("Request command: ",
			fmt.Sprintf("%s post an article", authorAccount.Name),
			" ",
			fmt.Sprintf("Result: %v", resp))
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

	signTx, err := utils.GenerateSignedTxAndValidate2(rpcClient, []interface{}{follow_op}, followerAccount, ChainId)
	if err != nil {
		fmt.Println(err)
		return
	}

	//fmt.Println("Request command: ", fmt.Sprintf("follow %s %s", followerAccount.Name, followingAccount.Name) )

	req := &grpcpb.BroadcastTrxRequest{Transaction: signTx}
	resp, err := rpcClient.BroadcastTrx(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		if strings.Contains(resp.Invoice.ErrorInfo,"net resource not enough") {
			stake(rpcClient,followerAccount,followerAccount,1)
		}
		fmt.Println("Request command: ",
			fmt.Sprintf("follow %s %s", followerAccount.Name, followingAccount.Name),
			" ",
			fmt.Sprintf("Result: %v", resp))
	}
}

func voteArticle(rpcClient grpcpb.ApiServiceClient, voterAccount *wallet.PrivAccount, postId uint64) {
	if voterAccount == nil {
		GlobalAccountLIst.RLock()
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		idx := r.Intn( len(GlobalAccountLIst.arr) )
		voterAccount = GlobalAccountLIst.arr[idx]
		GlobalAccountLIst.RUnlock()
	}

	if voterAccount.Name == "initminer" {
		return
	}

	if postId == 0 {
		PostIdList.RLock()
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		idx := r.Intn( len(PostIdList.arr) )
		postId = PostIdList.arr[idx]
		PostIdList.RUnlock()
	}

	vote_op := &prototype.VoteOperation{
		Voter: &prototype.AccountName{Value: voterAccount.Name},
		Idx:   postId,
	}

	signTx, err := utils.GenerateSignedTxAndValidate2(rpcClient, []interface{}{vote_op}, voterAccount, ChainId)
	if err != nil {
		fmt.Println(err)
		return
	}

	//fmt.Println("Request command: ", fmt.Sprintf("vote %s %d", voterAccount.Name, postId) )

	req := &grpcpb.BroadcastTrxRequest{Transaction: signTx}
	resp, err := rpcClient.BroadcastTrx(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		if strings.Contains(resp.Invoice.ErrorInfo, "Insufficient") {
			err := transfer(rpcClient, GlobalAccountLIst.arr[0], voterAccount, 5)
			if err != nil {
				fmt.Println(err)
				return
			}
			err = vest(rpcClient, GlobalAccountLIst.arr[0], voterAccount, 5)
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(fmt.Sprintf("====== vote from:%v to:%v amount:%v",GlobalAccountLIst.arr[0].Name,voterAccount.Name,5))
			voteArticle(rpcClient, voterAccount, postId)
			return
		}
		if strings.Contains(resp.Invoice.ErrorInfo,"net resource not enough") {
			stake(rpcClient,voterAccount,voterAccount,1)
		}

		fmt.Println("Request command: ",
			fmt.Sprintf("vote %s %d", voterAccount.Name, postId),
			" ",
			fmt.Sprintf("Result: %v", resp))
	}
}

func replyArticle(rpcClient grpcpb.ApiServiceClient, fromAccount *wallet.PrivAccount, postId uint64) {
	if fromAccount == nil {
		GlobalAccountLIst.RLock()
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		idx := r.Intn( len(GlobalAccountLIst.arr) )
		fromAccount = GlobalAccountLIst.arr[idx]
		GlobalAccountLIst.RUnlock()
	}

	if postId == 0 {
		PostIdList.RLock()
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		idx := r.Intn( len(PostIdList.arr) )
		postId = PostIdList.arr[idx]
		PostIdList.RUnlock()
	}

	var content = ""
	for i:=0;i<128;i++ {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		idx := r.Intn(len(nameLib))
		content += string(nameLib[idx])
	}

	uuid := utils.GenerateUUID(fromAccount.Name)
	beneficiaries := []*prototype.BeneficiaryRouteType{}

	reply_op := &prototype.ReplyOperation{
		Uuid:          uuid,
		Owner:         &prototype.AccountName{Value: fromAccount.Name},
		Content:       content,
		ParentUuid:    postId,
		Beneficiaries: beneficiaries,
	}

	signTx, err := utils.GenerateSignedTxAndValidate2(rpcClient, []interface{}{reply_op}, fromAccount, ChainId)
	if err != nil {
		fmt.Println(err)
		return
	}

	//fmt.Println("Request command: ", fmt.Sprintf("reply %s %d", fromAccount.Name, postId) )

	req := &grpcpb.BroadcastTrxRequest{Transaction: signTx}
	resp, err := rpcClient.BroadcastTrx(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		if strings.Contains(resp.Invoice.ErrorInfo,"net resource not enough") {
			stake(rpcClient,fromAccount,fromAccount,1)
		}
		fmt.Println("Request command: ",
			fmt.Sprintf("reply %s %d", fromAccount.Name, postId),
			" ",
			fmt.Sprintf("Result: %v", resp))
	}
}

func acquireTicket(rpcClient grpcpb.ApiServiceClient, fromAccount *wallet.PrivAccount) {
	if fromAccount == nil {
		GlobalAccountLIst.RLock()
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		idx := r.Intn( len(GlobalAccountLIst.arr) )
		fromAccount = GlobalAccountLIst.arr[idx]
		GlobalAccountLIst.RUnlock()
	}
	if fromAccount.Name == "initminer" {
		return
	}

	acquireOp := &prototype.AcquireTicketOperation{
		Account: &prototype.AccountName{Value: fromAccount.Name},
		Count: 1,
	}

	signTx, err := utils.GenerateSignedTxAndValidate2(rpcClient, []interface{}{acquireOp}, fromAccount, ChainId)
	if err != nil {
		fmt.Println(err)
		return
	}

	//fmt.Println("Request command: ", fmt.Sprintf("reply %s %d", fromAccount.Name, postId) )

	req := &grpcpb.BroadcastTrxRequest{Transaction: signTx}
	resp, err := rpcClient.BroadcastTrx(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		if strings.Contains(resp.Invoice.ErrorInfo, "Insufficient") {
			if fromAccount == GlobalAccountLIst.arr[0] {
				fmt.Println("Initminer has no money left")
				return
			}
			err := vest(rpcClient, GlobalAccountLIst.arr[0], fromAccount, 10 * constants.COSTokenDecimals)
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(fmt.Sprintf("====== vest from:%v to:%v amount:%v",GlobalAccountLIst.arr[0].Name,fromAccount.Name,fromAccount, 10 * constants.COSTokenDecimals))
			acquireTicket(rpcClient, fromAccount)
			return
		}

		if strings.Contains(resp.Invoice.ErrorInfo,"net resource not enough") {
			stake(rpcClient,fromAccount,fromAccount,1)
		}
		fmt.Println("Request command: ",
			fmt.Sprintf("ticket acquire %s %d", fromAccount.Name, 1),
			" ",
			fmt.Sprintf("Result: %v", resp))
	}
}

func voteByTicket(rpcClient grpcpb.ApiServiceClient, fromAccount *wallet.PrivAccount, postId uint64) {
	if fromAccount == nil {
		GlobalAccountLIst.RLock()
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		idx := r.Intn( len(GlobalAccountLIst.arr) )
		fromAccount = GlobalAccountLIst.arr[idx]
		GlobalAccountLIst.RUnlock()
	}

	if fromAccount.Name == "initminer" {
		return
	}
	if postId == 0 {
		PostIdList.RLock()
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		idx := r.Intn( len(PostIdList.arr) )
		postId = PostIdList.arr[idx]
		PostIdList.RUnlock()
	}

	voteByTicketOp := &prototype.VoteByTicketOperation{
		Account: &prototype.AccountName{Value: fromAccount.Name},
		Idx: postId,
		Count: 1,
	}

	signTx, err := utils.GenerateSignedTxAndValidate2(rpcClient, []interface{}{voteByTicketOp}, fromAccount, ChainId)
	if err != nil {
		fmt.Println(err)
		return
	}

	//fmt.Println("Request command: ", fmt.Sprintf("reply %s %d", fromAccount.Name, postId) )

	req := &grpcpb.BroadcastTrxRequest{Transaction: signTx}
	resp, err := rpcClient.BroadcastTrx(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		if strings.Contains(resp.Invoice.ErrorInfo, "insufficient") {
			if fromAccount == GlobalAccountLIst.arr[0] {
				fmt.Println("Initminer has no money left")
				return
			}
			acquireTicket(rpcClient, fromAccount)
			fmt.Println(fmt.Sprintf("====== acquire ticket from:%v count:%v",fromAccount.Name,1))
			voteByTicket(rpcClient, fromAccount, postId)
			return
		}

		if strings.Contains(resp.Invoice.ErrorInfo,"net resource not enough") {
			stake(rpcClient,fromAccount,fromAccount,1)
		}
		fmt.Println("Request command: ",
			fmt.Sprintf("ticket vote %s %d %d", fromAccount.Name, postId, 1),
			" ",
			fmt.Sprintf("Result: %v", resp))
	}
}

func callContract(rpcClient grpcpb.ApiServiceClient, fromAccount  *wallet.PrivAccount) error {
	if fromAccount == nil {
		GlobalAccountLIst.RLock()
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		idx := r.Intn( len(GlobalAccountLIst.arr) )
		fromAccount = GlobalAccountLIst.arr[idx]
		GlobalAccountLIst.RUnlock()
	}

	param := fmt.Sprintf(" [\"%v\"] ", fromAccount.Name)

	applyOp := &prototype.ContractApplyOperation{
		Caller:   &prototype.AccountName{Value: fromAccount.Name},
		Owner:    &prototype.AccountName{Value: "initminer"},
		Amount:   &prototype.Coin{Value: 0},
		Contract: "PGRegister",
		Params:   param,
		Method:   "checkincount",
	}

	signTx, err := utils.GenerateSignedTxAndValidate2(rpcClient, []interface{}{applyOp}, fromAccount, ChainId)
	if err != nil {
		fmt.Println(err)
		return err
	}

	req := &grpcpb.BroadcastTrxRequest{Transaction: signTx}
	resp, err := rpcClient.BroadcastTrx(context.Background(), req)
	if err != nil {
		fmt.Println(err)
		return err
	} else {
		if strings.Contains(resp.Invoice.ErrorInfo,"net resource not enough") {
			stake(rpcClient,fromAccount,fromAccount,1)
		}
		fmt.Println("Request command: ",
			fmt.Sprintf("callContract %s %s %d", fromAccount.Name),
			" ",
			fmt.Sprintf("Result: %v", resp))
	}
	return nil
}

func RandomUnRegisterBP(rpcClient grpcpb.ApiServiceClient) error {

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	index := r.Intn( len(BPList) )

	bpAccount := &wallet.PrivAccount{
		Account: wallet.Account{Name: BPList[index].name, PubKey: BPList[index].pubKeyStr},
		PrivKey: BPList[index].priKeyStr,
	}

	bpUnregister_op := &prototype.BpUnregisterOperation{
		Owner: &prototype.AccountName{Value: BPList[index].name},
	}

	signTx, err := utils.GenerateSignedTxAndValidate2(rpcClient, []interface{}{bpUnregister_op}, bpAccount, ChainId)
	if err != nil {
		fmt.Println(err)
		return err
	}
	req := &grpcpb.BroadcastTrxRequest{Transaction: signTx}
	resp, err := rpcClient.BroadcastTrx(context.Background(), req)
	if err == nil {
		if resp.Invoice.Status == 200 {
			lastConductBPIndex = index
		}
		if strings.Contains(resp.Invoice.ErrorInfo,"net resource not enough") {
			stake(rpcClient, bpAccount, bpAccount,1)
		}
		fmt.Println("Request command: ",
			fmt.Sprintf("unregister bp %s", BPList[index].name),
			" ",
			fmt.Sprintf("Result: %v", resp))
	}
	return err
}

func RegisterAndVoteBP(rpcClient grpcpb.ApiServiceClient, index int) error {
	resp, _ := rpcClient.GetChainState( context.Background(), &grpcpb.NonParamsRequest{} )
	refBlockPrefix := common.TaposRefBlockPrefix(resp.State.Dgpo.HeadBlockId.Hash)
	refBlockNum := common.TaposRefBlockNum(resp.State.Dgpo.HeadBlockNumber)
	tx := &prototype.Transaction{RefBlockNum: refBlockNum, RefBlockPrefix: refBlockPrefix, Expiration: &prototype.TimePointSec{UtcSeconds: resp.State.Dgpo.Time.UtcSeconds + 30}}
	trx := &prototype.SignedTransaction{Trx: tx}

	pubKey, err := prototype.PublicKeyFromWIF(BPList[index].pubKeyStr)
	if err != nil {
		return err
	}

	opBpReg := &prototype.BpRegisterOperation{
		Owner:           &prototype.AccountName{Value: BPList[index].name},
		Url:             BPList[index].name,
		Desc:            BPList[index].name,
		BlockSigningKey: pubKey,
		Props: &prototype.ChainProperties{
			AccountCreationFee:    prototype.NewCoin(constants.DefaultAccountCreateFee),
			MaximumBlockSize:      10 * 1024 * 1024,
			StaminaFree:           constants.DefaultStaminaFree,
			TpsExpected:           constants.DefaultTPSExpected,
			EpochDuration:         constants.InitEpochDuration,
			TopNAcquireFreeToken:  constants.InitTopN,
			PerTicketPrice:        prototype.NewCoin(constants.PerTicketPrice * constants.COSTokenDecimals),
			PerTicketWeight:       constants.PerTicketWeight,
		},
	}

	opBpVote := &prototype.BpVoteOperation{
		Voter: prototype.NewAccountName(BPList[index].name),
		Witness: prototype.NewAccountName(BPList[index].name),
		Cancel: false}

	trx.Trx.AddOperation(opBpReg)
	trx.Trx.AddOperation(opBpVote)

	keys, err := prototype.PrivateKeyFromWIF(BPList[index].priKeyStr)
	if err != nil {
		return err
	}
	res := trx.Sign(keys, ChainId)
	trx.Signature = &prototype.SignatureType{Sig: res}

	if err := trx.Validate(); err != nil {
		return err
	}

	req := &grpcpb.BroadcastTrxRequest{Transaction: trx}
	newresp, err := rpcClient.BroadcastTrx(context.Background(), req)

	if err == nil {
		if newresp.Invoice.Status == 200 {
			lastConductBPIndex = -1
		}
		if strings.Contains(newresp.Invoice.ErrorInfo,"net resource not enough") {
			bpAccount := &wallet.PrivAccount{
				Account: wallet.Account{Name: BPList[index].name, PubKey: BPList[index].pubKeyStr},
				PrivKey: BPList[index].priKeyStr,
			}
			stake(rpcClient, bpAccount, bpAccount,1)
		}
		fmt.Println("Request command: ",
			fmt.Sprintf("register bp and vote himself %s", BPList[index].name),
			" ",
			fmt.Sprintf("Result: %v", newresp))
	}

	return err
}

func getBPListOnChain(rpcClient grpcpb.ApiServiceClient) (bpList *grpcpb.GetWitnessListResponse, err error) {
	req := &grpcpb.GetWitnessListByVoteCountRequest{}
	req.Limit = uint32(len(BPList))
	resp, err := rpcClient.GetWitnessListByVoteCount(context.Background(), req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}