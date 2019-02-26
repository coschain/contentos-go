package app

import (
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/prototype"
	"github.com/golang/protobuf/proto"
	"io/ioutil"
	"testing"
)

const (
	accountNameBob = "bobob"
	pubKeyBob      = "COS6oLVaFEtHZmPDuCvuB48NpSKytjyavPk5MwtN4HqKG16oSA2wS"
	priKeyBob      = "EpgwWxboEdaWfEBdWswobsBt8pBF6xoYQPayBs4eVysMGGGYL"

	accountNameTom = "tomom"
	pubKeyTom      = "COS5LgGC16xurDrmfC7Yv5RGUeWeCPUP4tdW627vqXk9eQ97ZEJ7P"
	priKeyTom      = "aFovWd8qS1yUAr94ULbG6ASwUsfPS3GX1ebPGDzowrUxQp1ta"
)

func makeBlock(pre *prototype.Sha256, blockTimestamp uint32, signedTrx *prototype.SignedTransaction) *prototype.SignedBlock {
	sigBlk := new(prototype.SignedBlock)

	// add trx wraper
	trxWraper := &prototype.TransactionWrapper{
		SigTrx:  signedTrx,
		Invoice: &prototype.TransactionReceipt{Status: prototype.StatusSuccess},
	}
	sigBlk.Transactions = append(sigBlk.Transactions, trxWraper)

	// calculate merkle
	id := sigBlk.CalculateMerkleRoot()

	// write signed block header
	sigBlkHdr := new(prototype.SignedBlockHeader)

	sigBlkHdr.Header = new(prototype.BlockHeader)
	sigBlkHdr.Header.Previous = pre
	sigBlkHdr.Header.Timestamp = &prototype.TimePointSec{UtcSeconds: blockTimestamp}
	sigBlkHdr.Header.Witness = &prototype.AccountName{Value: constants.INIT_MINER_NAME}
	sigBlkHdr.Header.TransactionMerkleRoot = &prototype.Sha256{Hash: id.Data[:]}
	sigBlkHdr.WitnessSignature = &prototype.SignatureType{}

	// signature
	pri, err := prototype.PrivateKeyFromWIF(constants.INITMINER_PRIKEY)
	if err != nil {
		panic("PrivateKeyFromWIF error")
	}
	sigBlkHdr.Sign(pri)

	sigBlk.SignedHeader = sigBlkHdr
	return sigBlk
}

func createSigTrx(ops []interface{}, c *TrxPool,priKey string) (*prototype.SignedTransaction, error) {

	headBlockID := c.GetProps().GetHeadBlockId()
	expire := c.GetProps().Time.UtcSeconds + 20;
	expire += 20;

	privKey, err := prototype.PrivateKeyFromWIF(priKey)
	if err != nil {
		return nil, err
	}

	tx := &prototype.Transaction{RefBlockNum: 0, RefBlockPrefix: 0,
		Expiration: &prototype.TimePointSec{UtcSeconds: expire}}
	for _,op := range ops {
		tx.AddOperation(op)
	}

	// set reference
	id := &common.BlockID{}
	copy(id.Data[:], headBlockID.Hash[:])
	tx.SetReferenceBlock(id)

	signTx := prototype.SignedTransaction{Trx: tx}

	res := signTx.Sign(privKey, prototype.ChainId{Value: 0})
	signTx.Signatures = append(signTx.Signatures, &prototype.SignatureType{Sig: res})

	return &signTx, nil
}

func makeCreateAccountOP(accountName string, pubKey string) (*prototype.AccountCreateOperation, error) {
	pub, err := prototype.PublicKeyFromWIF(pubKey)
	if err != nil {
		return nil, errors.New("PublicKeyFromWIF error")
	}
	acop := &prototype.AccountCreateOperation{
		Fee:            prototype.NewCoin(1),
		Creator:        &prototype.AccountName{Value: constants.COS_INIT_MINER},
		NewAccountName: &prototype.AccountName{Value: accountName},
		Owner: &prototype.Authority{
			WeightThreshold: 1,
			AccountAuths: []*prototype.KvAccountAuth{
				&prototype.KvAccountAuth{
					Name:   &prototype.AccountName{Value: constants.COS_INIT_MINER},
					Weight: 3,
				},
			},
			KeyAuths: []*prototype.KvKeyAuth{
				&prototype.KvKeyAuth{
					Key:    pub, // owner key
					Weight: 1,
				},
			},
		},
	}

	return acop, nil
}

func Test_PushTrx(t *testing.T) {
	// set up controller
	db := startDB()
	defer clearDB(db)
	c := startController(db)

	acop, err := makeCreateAccountOP(accountNameBob, pubKeyBob)
	if err != nil {
		t.Error("makeCreateAccountOP error:", err)
	}
	ops := []interface{}{}
	ops = append(ops,acop)

	signedTrx, err := createSigTrx(ops, c, constants.INITMINER_PRIKEY)
	if err != nil {
		t.Error("createSigTrx error:", err)
	}

	invoice := c.PushTrx(signedTrx)
	if invoice.Status != prototype.StatusSuccess {
		t.Error("PushTrx return status error:", invoice.Status)
	}

	bobName := &prototype.AccountName{Value: accountNameBob}
	bobWrap := table.NewSoAccountWrap(db, bobName)
	if !bobWrap.CheckExist() {
		t.Error("create account failed")
	}
}

func Test_PushBlock(t *testing.T) {

	// set up controller
	db := startDB()
	defer clearDB(db)
	c := startController(db)

	createOP, err := makeCreateAccountOP(accountNameBob, pubKeyBob)
	if err != nil {
		t.Error("makeCreateAccountOP error:", err)
	}

	ops := []interface{}{}
	ops = append(ops,createOP)
	signedTrx, err := createSigTrx(ops, c, constants.INITMINER_PRIKEY)
	if err != nil {
		t.Error("createSigTrx error:", err)
	}

	sigBlk := makeBlock(c.GetProps().GetHeadBlockId(), 10, signedTrx)

	fmt.Println("block size:", proto.Size(sigBlk))

	c.PushBlock(sigBlk, prototype.Skip_nothing)

	bobName := &prototype.AccountName{Value: accountNameBob}
	bobWrap := table.NewSoAccountWrap(db, bobName)
	if !bobWrap.CheckExist() {
		t.Error("create account failed")
	}
}

func TestController_GenerateAndApplyBlock(t *testing.T) {
	createOP, err := makeCreateAccountOP(accountNameBob, pubKeyBob)
	if err != nil {
		t.Error("makeCreateAccountOP error:", err)
	}
	// set up controller
	db := startDB()
	defer clearDB(db)
	c := startController(db)

	ops := []interface{}{}
	ops = append(ops,createOP)
	signedTrx, err := createSigTrx(ops, c, constants.INITMINER_PRIKEY)
	if err != nil {
		t.Error("createSigTrx error:", err)
	}

	invoice := c.PushTrx(signedTrx)
	if invoice.Status != prototype.StatusSuccess {
		t.Error("PushTrx return status error:", invoice.Status)
	}

	bobName := &prototype.AccountName{Value: accountNameBob}
	bobWrap := table.NewSoAccountWrap(db, bobName)
	if !bobWrap.CheckExist() {
		t.Error("create account failed")
	}

	pri, err := prototype.PrivateKeyFromWIF(constants.INITMINER_PRIKEY)
	if err != nil {
		t.Error("PrivateKeyFromWIF error")
	}

	pre := &prototype.Sha256{Hash: make([]byte,32)}
	block,err := c.GenerateAndApplyBlock(constants.INIT_MINER_NAME, pre, 18, pri, 0)
	dgpWrap := table.NewSoGlobalWrap(db,&SingleId)
	mustSuccess(block.Id().BlockNum() == dgpWrap.GetProps().HeadBlockNumber,"block number error",prototype.StatusError)
	bobWrap2 := table.NewSoAccountWrap(db, bobName)
	if !bobWrap2.CheckExist() {
		t.Error("create account failed")
	}
}

func Test_list(t *testing.T) {

	// set up controller
	db := startDB()
	defer clearDB(db)
	c := startController(db)

	// make trx
	acop, err := makeCreateAccountOP(accountNameBob, pubKeyBob)
	if err != nil {
		t.Error("makeCreateAccountOP error:", err)
	}

	ops := []interface{}{}
	ops = append(ops,acop)
	signedTrx, err := createSigTrx(ops, c, constants.INITMINER_PRIKEY)
	if err != nil {
		t.Error("createSigTrx error:", err)
	}
	id, err := signedTrx.Id()

	// insert trx into DB unique table
	transactionObjWrap := table.NewSoTransactionObjectWrap(db, id)
	if transactionObjWrap.CheckExist() {
		panic("Duplicate transaction check failed")
	}

	cErr := transactionObjWrap.Create(func(tInfo *table.SoTransactionObject) {
		tInfo.TrxId = id
		tInfo.Expiration = &prototype.TimePointSec{UtcSeconds: 100}
	})
	if cErr != nil {
		panic("create transactionObject failed")
	}

	// check and delete

	sortWrap := table.STransactionObjectExpirationWrap{Dba: db}
	sortWrap.ForEachByOrder(nil, nil,
		func(mVal *prototype.Sha256, sVal *prototype.TimePointSec, idx uint32) bool {
		   if sVal != nil {
			   objWrap := table.NewSoTransactionObjectWrap(db, mVal)
			   if !objWrap.RemoveTransactionObject() {
				   panic("RemoveTransactionObject error")
			   }
		   }
		   return true
	})
}

func TestController_GetWitnessTopN(t *testing.T) {

	// set up controller
	db := startDB()
	defer clearDB(db)
	c := startController(db)

	name := &prototype.AccountName{Value: "wit1"}
	witnessWrap := table.NewSoWitnessWrap(db, name)
	mustNoError(witnessWrap.Create(func(tInfo *table.SoWitness) {
		tInfo.Owner = name
		tInfo.WitnessScheduleType = &prototype.WitnessScheduleType{Value: prototype.WitnessScheduleType_miner}
		tInfo.CreatedTime = &prototype.TimePointSec{UtcSeconds: 0}
		tInfo.SigningKey = &prototype.PublicKeyType{Data: []byte{1}}
		tInfo.LastWork = &prototype.Sha256{Hash: []byte{0}}
	}), "Witness Create Error",prototype.StatusError)

	name2 := &prototype.AccountName{Value: "wit2"}
	witnessWrap2 := table.NewSoWitnessWrap(db, name2)
	mustNoError(witnessWrap2.Create(func(tInfo *table.SoWitness) {
		tInfo.Owner = name2
		tInfo.WitnessScheduleType = &prototype.WitnessScheduleType{Value: prototype.WitnessScheduleType_miner}
		tInfo.CreatedTime = &prototype.TimePointSec{UtcSeconds: 0}
		tInfo.SigningKey = &prototype.PublicKeyType{Data: []byte{2}}
		tInfo.LastWork = &prototype.Sha256{Hash: []byte{0}}
	}), "Witness Create Error",prototype.StatusError)

	witnesses := c.GetWitnessTopN(10)

	for _, wit := range witnesses {
		fmt.Println(wit)
	}
}

func TestController_PopBlock(t *testing.T) {

	// set up controller
	db := startDB()
	defer clearDB(db)
	c := startController(db)

	createOP, err := makeCreateAccountOP(accountNameBob, pubKeyBob)
	if err != nil {
		t.Error("makeCreateAccountOP error:", err)
	}

	ops := []interface{}{}
	ops = append(ops,createOP)
	signedTrx, err := createSigTrx(ops, c, constants.INITMINER_PRIKEY)
	if err != nil {
		t.Error("createSigTrx error:", err)
	}

	block := makeBlock(c.GetProps().GetHeadBlockId(), 6, signedTrx)

	fmt.Println("block size:", proto.Size(block))

	c.PushBlock(block, prototype.Skip_nothing)

	// second block
	createOP2, err := makeCreateAccountOP(accountNameTom, pubKeyTom)
	if err != nil {
		t.Error("makeCreateAccountOP error:", err)
	}

	ops[0] = createOP2
	signedTrx2, err := createSigTrx(ops, c, constants.INITMINER_PRIKEY)
	if err != nil {
		t.Error("createSigTrx error:", err)
	}

	block2 := makeBlock(c.GetProps().GetHeadBlockId(), 9, signedTrx2)

	c.PushBlock(block2, prototype.Skip_nothing)

	// check
	bobName := &prototype.AccountName{Value: accountNameBob}
	bobWrap := table.NewSoAccountWrap(db, bobName)
	if !bobWrap.CheckExist() {
		t.Error("create account failed")
	}
	tomName := &prototype.AccountName{Value: accountNameTom}
	tomWrap := table.NewSoAccountWrap(db, tomName)
	if !tomWrap.CheckExist() {
		t.Error("create account failed")
	}


	c.PopBlock(block2.Id().BlockNum())
	tomNoExistWrap := table.NewSoAccountWrap(db, tomName)
	if tomNoExistWrap.CheckExist() || c.GetProps().HeadBlockNumber != 1 { // need check c.dgpo.HeadBlockNumber
		t.Error("pop block error")
	}

	c.PopBlock(block.Id().BlockNum())
	bobNoExistWrap := table.NewSoAccountWrap(db, bobName)
	if bobNoExistWrap.CheckExist() || c.GetProps().HeadBlockNumber != 0 { // need check c.dgpo.HeadBlockNumber
		t.Error("pop block error")
	}

}

func TestController_Commit(t *testing.T) {

	// set up controller
	db := startDB()
	defer clearDB(db)
	c := startController(db)

	createOP, err := makeCreateAccountOP(accountNameBob, pubKeyBob)
	if err != nil {
		t.Error("makeCreateAccountOP error:", err)
	}

	ops := []interface{}{}
	ops = append(ops,createOP)
	signedTrx, err := createSigTrx(ops, c,constants.INITMINER_PRIKEY)
	if err != nil {
		t.Error("createSigTrx error:", err)
	}

	block := makeBlock(c.GetProps().GetHeadBlockId(), 6, signedTrx)

	fmt.Println("block size:", proto.Size(block))

	c.PushBlock(block, prototype.Skip_nothing)

	// second block
	createOP2, err := makeCreateAccountOP(accountNameTom, pubKeyTom)
	if err != nil {
		t.Error("makeCreateAccountOP error:", err)
	}

	ops[0] = createOP2
	signedTrx2, err := createSigTrx(ops, c, constants.INITMINER_PRIKEY)
	if err != nil {
		t.Error("createSigTrx error:", err)
	}

	block2 := makeBlock(c.GetProps().GetHeadBlockId(), 9, signedTrx2)

	c.PushBlock(block2, prototype.Skip_nothing)

	// check
	bobName := &prototype.AccountName{Value: accountNameBob}
	bobWrap := table.NewSoAccountWrap(db, bobName)
	if !bobWrap.CheckExist() {
		t.Error("create account failed")
	}
	tomName := &prototype.AccountName{Value: accountNameTom}
	tomWrap := table.NewSoAccountWrap(db, tomName)
	if !tomWrap.CheckExist() {
		t.Error("create account failed")
	}

	c.Commit(2)
	bobStillExistWrap := table.NewSoAccountWrap(db, bobName)
	if !bobStillExistWrap.CheckExist() {
		t.Error("commit error")
	}

	tomStillExistWrap := table.NewSoAccountWrap(db, tomName)
	if !tomStillExistWrap.CheckExist() {
		t.Error("commit error")
	}

	defer func() {
		if err := recover(); err == nil {
			t.Error("pop a irreversible block but no panic")
		}
	}()
	c.PopBlock(1)
}

func Test_MixOp(t *testing.T) {

	db := startDB()
	defer clearDB(db)
	c := startController(db)

	// deploy contract
	data, _ := ioutil.ReadFile("./test_data/hello.wasm")
	abi, _ := ioutil.ReadFile("./test_data/hello.abi")
	deployOp := &prototype.ContractDeployOperation{
		Owner:    &prototype.AccountName{Value: "initminer"},
		Contract: "hello",
		Abi:      string(abi),
		Code:     data,
	}
	ops := []interface{}{}
	ops = append(ops,deployOp)

	signedTrx, err := createSigTrx(ops, c, constants.INITMINER_PRIKEY)
	if err != nil {
		t.Error("createSigTrx error:", err)
	}

	invoice := c.PushTrx(signedTrx)
	if invoice.Status != prototype.StatusSuccess {
		t.Error("PushTrx return status error:", invoice.Status)
	}

	// first op : call contract
	applyOp := &prototype.ContractApplyOperation{
		Caller:   &prototype.AccountName{Value: "initminer"},
		Owner:    &prototype.AccountName{Value: "initminer"},
		Contract: "hello",
		Method: "hi",
		Params: "[\"contentos\"]",
		//Amount:   &prototype.Coin{Value: 1000},
		Gas:      &prototype.Coin{Value: 300000},
	}

	ops = ops[:0]
	ops = append(ops,applyOp)

	//
	miner := &prototype.AccountName{Value: "initminer"}
	minerWrap := table.NewSoAccountWrap(db, miner)
	b := minerWrap.GetStamina()
	t.Log("before initminer stamina:",b)
	//

	const value = 1000000000
	// second op : transfer to a invalid account, should failed
	transOp := &prototype.TransferOperation{
		From:   &prototype.AccountName{Value: "initminer"},
		To:     &prototype.AccountName{Value: "someone"},
		Amount: prototype.NewCoin(value),
	}
	ops = append(ops,transOp)

	signedTrx2, err := createSigTrx(ops, c, constants.INITMINER_PRIKEY)
	if err != nil {
		t.Error("createSigTrx error:", err)
	}

	invoice2 := c.PushTrx(signedTrx2)
	if invoice2.Status != prototype.StatusSuccess &&  invoice2.Status != prototype.StatusDeductGas{
		t.Error("PushTrx return status error:", invoice2.Status)
	}

	//
	minerWrap2 := table.NewSoAccountWrap(db, miner)
	b2 := minerWrap2.GetStamina()
	t.Log("after initminer stamina:",b2)
	//

	// right result:
	// 1. gas should be deduct
	// 2. transfer should be revert
	if b > b2{
		t.Error("gas error or db error")
	}
}

func Test_Stake(t *testing.T) {
	db := startDB()
	defer clearDB(db)
	c := startController(db)

	wraper := table.NewSoAccountWrap(db,prototype.NewAccountName(constants.COS_INIT_MINER))
	if wraper.GetStakeVesting().Value != 0 {
		t.Error("stake vesting error")
	}

	stakeOp := &prototype.StakeOperation{
		Account:prototype.NewAccountName(constants.COS_INIT_MINER),
		Amount:100,
	}
	ops := []interface{}{}
	ops = append(ops,stakeOp)

	signedTrx, err := createSigTrx(ops, c, constants.INITMINER_PRIKEY)
	if err != nil {
		t.Error("createSigTrx error:", err)
	}

	invoice := c.PushTrx(signedTrx)
	if invoice.Status != prototype.StatusSuccess {
		t.Error("PushTrx return status error:", invoice.Status)
	}

	if wraper.GetStakeVesting().Value == 0 {
		t.Error("stake vesting error")
	}
}
