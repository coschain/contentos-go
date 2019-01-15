package commands

import (
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"time"
)

func createAndTransfer () {
	StartNode()
	defer clearAll()

	time.Sleep(10 * time.Second)
	for i:=0;i<len(globalObj.dposList);i++ {
		fmt.Println()
		fmt.Println()
		fmt.Println("main func active producers:   ", globalObj.dposList[i].ActiveProducers())
		fmt.Println()
		fmt.Println()
	}

	prodTime := time.Now()
	count := 8

	// produce count blocks and check every chain's head block number, the number should be equal to count
	for i:=0;i<count;i++ {
		prodTime = prodTime.Add( time.Duration( 3 * i ) * time.Second )
		produceBlk(globalObj.dposList[0], prodTime)

	}
	time.Sleep(5 * time.Second)
	for i:=0;i<int(NodeCnt);i++ {
		fmt.Println("head block id:   ", globalObj.dposList[i].GetHeadBlockId())
		if int( globalObj.dposList[i].GetHeadBlockId().Data[0] ) != count {
			panic(errors.New("head block number mismached"))
		}
	}

	// check every chain, initminer1, initminer2......, these accounts should not exist
	for i:=0;i<int(NodeCnt);i++ {
		for j:=1;j<int(NodeCnt);j++ {
			acc := getAccount(globalObj.dbList[i], fmt.Sprintf("initminer%d", j))
			if acc != nil {
				panic(errors.New("this account should not exist"))
			}
		}
	}

	// create account initminer1, initminer2,......, and then check every chain whether these accounts exist
	for i:=1;i<int(NodeCnt);i++ {
		createAccount(globalObj.dposList[i],  fmt.Sprintf("initminer%d", i))
	}
	prodTime = prodTime.Add(3 * time.Second)
	produceBlk(globalObj.dposList[0], prodTime)
	time.Sleep(5 * time.Second)
	for i:=0;i<int(NodeCnt);i++ {
		for j:=1;j<int(NodeCnt);j++ {
			acc := getAccount(globalObj.dbList[i], fmt.Sprintf("initminer%d", j))
			if acc == nil {
				panic(errors.New("account should exist"))
			}
		}
	}

	// transfer 10 coin from initminer to initminer1, initminer2......, and then check every chain the account's value
	for i:=1;i<int(NodeCnt);i++ {
		transfer(globalObj.dposList[i], "initminer", fmt.Sprintf("initminer%d", i), 10 , "")
	}
	prodTime = prodTime.Add(3 * time.Second)
	produceBlk(globalObj.dposList[0], prodTime)
	time.Sleep(5 * time.Second)
	for i:=0;i<int(NodeCnt);i++ {
		for j:=1;j<int(NodeCnt);j++ {
			acc := getAccount(globalObj.dbList[i], fmt.Sprintf("initminer%d", j))
			if acc.GetBalance().Value != 10 {
				panic(errors.New("account value mismached"))
			}
		}
	}
}

func produceBlk (icons iservices.IConsensus, t time.Time) {
	icons.ResetTicker(t)
	icons.MaybeProduceBlock()
}

func getAccount(idb iservices.IDatabaseService, name string) *table.SoAccountWrap {
	accWrap := table.NewSoAccountWrap(idb, &prototype.AccountName{Value: name})
	if !accWrap.CheckExist() {
		return nil
	}
	return accWrap
}

func createAccount(icons iservices.IConsensus, name string) {
	defaultPrivKey, err := prototype.PrivateKeyFromWIF(constants.INITMINER_PRIKEY)
	if err != nil {
		panic(err)
	}
	defaultPubKey, err := defaultPrivKey.PubKey()
	if err != nil {
		panic(err)
	}

	keys := prototype.NewAuthorityFromPubKey(defaultPubKey)

	// create account with default pub key
	acop := &prototype.AccountCreateOperation{
		Fee:            prototype.NewCoin(1),
		Creator:        &prototype.AccountName{Value: "initminer"},
		NewAccountName: &prototype.AccountName{Value: name},
		Owner:          keys,
	}
	// use initminer's priv key sign
	signTx, err := signTrx(icons, defaultPrivKey.ToWIF(), acop)
	if err != nil {
		panic(err)
	}
	icons.PushTransaction(signTx, true, true)
	time.Sleep(3 * time.Second)
}

func signTrx(icons iservices.IConsensus, privKeyStr string, ops ...interface{}) (*prototype.SignedTransaction, error) {
	privKey, err := prototype.PrivateKeyFromWIF(privKeyStr)
	if err != nil {
		return nil, err
	}
	headBlockID := icons.GetHeadBlockId()
	headBlk, err := icons.FetchBlock(headBlockID)
	if err != nil {
		panic(err)
	}
	tx := &prototype.Transaction{RefBlockNum: 0, RefBlockPrefix: 0, Expiration: &prototype.TimePointSec{UtcSeconds: uint32(headBlk.Timestamp() + constants.TRX_MAX_EXPIRATION_TIME)}}
	id := &common.BlockID{}
	id = &headBlockID
	tx.SetReferenceBlock(id)
	for _, op := range ops {
		tx.AddOperation(op)
	}
	signTx := prototype.SignedTransaction{Trx: tx}
	res := signTx.Sign(privKey, prototype.ChainId{Value: 0})
	signTx.Signatures = append(signTx.Signatures, &prototype.SignatureType{Sig: res})
	if err := signTx.Validate(); err != nil {
		return nil, err
	}
	return &signTx, nil
}

func transfer(icons iservices.IConsensus, from, to string, amount uint64, memo string) {
	defaultPrivKey, err := prototype.PrivateKeyFromWIF(constants.INITMINER_PRIKEY)
	if err != nil {
		panic(err)
	}
	top := &prototype.TransferOperation{
		From:   &prototype.AccountName{Value: from},
		To:     &prototype.AccountName{Value: to},
		Amount: prototype.NewCoin(amount),
		Memo:   memo,
	}
	var signTx *prototype.SignedTransaction
	signTx, err = signTrx(icons, defaultPrivKey.ToWIF(), top)
	if err != nil {
		panic(err)
	}
	icons.PushTransaction(signTx, true, true)
	time.Sleep(3 * time.Second)
}