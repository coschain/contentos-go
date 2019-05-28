package app

import (
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"github.com/golang/protobuf/proto"
	"io/ioutil"
	"testing"
	"time"
)

const (
	accountNameBob = "mynameisbob"
	pubKeyBob      = "COS5Tm9D28Mz8jUf8wwg8FGY7q2bnZ91aZRjzhPdrY738DBeou3v5"
	priKeyBob      = "47o8DKDKkRqLfM1HCPzcSYja5N5Z8PhmZYXGTo1pPrseJjZyM9"

	accountNameTom = "mynameistom"
	pubKeyTom      = "COS5E2vDnf245ydZBBUgQ8RkjNBzoKvGyr9kW4rfMMQcnkiD8JnEd"
	priKeyTom      = "3u6RCpDUEEUmB9QsFMNKCfEY54WWtmcXvqyD2NcHCDzhuhrP8F"
)

func addGlobalTime(db iservices.IDatabaseService, delta uint32) {
	wrap := table.NewSoGlobalWrap(db, &constants.GlobalId)
	gp := wrap.GetProps()
	gp.Time.UtcSeconds += delta
	wrap.MdProps(gp)
}

func addGlobalHeadNumer(db iservices.IDatabaseService, delta uint64) {
	wrap := table.NewSoGlobalWrap(db, &constants.GlobalId)
	gp := wrap.GetProps()
	gp.HeadBlockNumber += delta
	wrap.MdProps(gp)
}

func createSigTrxTmp(c *TrxPool, priKey string, step uint32, ops ...interface{}) (*prototype.SignedTransaction, error) {

	headBlockID := c.GetProps().GetHeadBlockId()
	expire := c.GetProps().Time.UtcSeconds
	expire += step

	privKey, err := prototype.PrivateKeyFromWIF(priKey)
	if err != nil {
		return nil, err
	}

	wrap := table.NewSoGlobalWrap(c.db, &constants.GlobalId)

	tx := &prototype.Transaction{RefBlockNum: uint32(wrap.GetProps().HeadBlockNumber), RefBlockPrefix: wrap.GetProps().HeadBlockPrefix,
		Expiration: &prototype.TimePointSec{UtcSeconds: expire}}
	for _, op := range ops {
		tx.AddOperation(op)
	}

	// set reference
	id := &common.BlockID{}
	copy(id.Data[:], headBlockID.Hash[:])
	tx.SetReferenceBlock(id)

	signTx := prototype.SignedTransaction{Trx: tx}

	res := signTx.Sign(privKey, prototype.ChainId{Value: 0})
	signTx.Signature = &prototype.SignatureType{Sig: res}

	return &signTx, nil
}

func makeBlockWithCommonTrx(pre *prototype.Sha256, blockTimestamp uint32, signedTrx *prototype.SignedTransaction) *prototype.SignedBlock {
	sigBlk := new(prototype.SignedBlock)

	// add trx wraper
	trxWraper := &prototype.TransactionWrapper{
		SigTrx:  signedTrx,
		Receipt: &prototype.TransactionReceipt{Status: prototype.StatusSuccess},
	}
	trxWraper.Receipt.NetUsage = uint64(proto.Size(signedTrx) * int(float64(constants.NetConsumePointNum)/float64(constants.NetConsumePointDen)))
	trxWraper.Receipt.CpuUsage = 1 // now we set 1 stamina for common operation
	sigBlk.Transactions = append(sigBlk.Transactions, trxWraper)

	// calculate merkle
	id := sigBlk.CalculateMerkleRoot()

	// write signed block header
	sigBlkHdr := new(prototype.SignedBlockHeader)

	sigBlkHdr.Header = new(prototype.BlockHeader)
	sigBlkHdr.Header.Previous = pre
	sigBlkHdr.Header.Timestamp = &prototype.TimePointSec{UtcSeconds: blockTimestamp}
	sigBlkHdr.Header.Witness = &prototype.AccountName{Value: constants.COSInitMiner}
	sigBlkHdr.Header.TransactionMerkleRoot = &prototype.Sha256{Hash: id.Data[:]}
	sigBlkHdr.WitnessSignature = &prototype.SignatureType{}

	// signature
	pri, err := prototype.PrivateKeyFromWIF(constants.InitminerPrivKey)
	if err != nil {
		panic("PrivateKeyFromWIF error")
	}
	sigBlkHdr.Sign(pri)

	sigBlk.SignedHeader = sigBlkHdr
	return sigBlk
}

func createSigTrx(c *TrxPool, priKey string, ops ...interface{}) (*prototype.SignedTransaction, error) {

	headBlockID := c.GetProps().GetHeadBlockId()
	expire := c.GetProps().Time.UtcSeconds + 20

	privKey, err := prototype.PrivateKeyFromWIF(priKey)
	if err != nil {
		return nil, err
	}

	tx := &prototype.Transaction{RefBlockNum: 0, RefBlockPrefix: 0,
		Expiration: &prototype.TimePointSec{UtcSeconds: expire}}
	for _, op := range ops {
		tx.AddOperation(op)
	}

	// set reference
	id := &common.BlockID{}
	copy(id.Data[:], headBlockID.Hash[:])
	tx.SetReferenceBlock(id)

	signTx := prototype.SignedTransaction{Trx: tx}

	res := signTx.Sign(privKey, prototype.ChainId{Value: 0})
	signTx.Signature = &prototype.SignatureType{Sig: res}

	return &signTx, nil
}

func makeCreateAccountOP(accountName string, pubKey string) (*prototype.AccountCreateOperation, error) {
	pub, err := prototype.PublicKeyFromWIF(pubKey)
	if err != nil {
		return nil, errors.New("PublicKeyFromWIF error")
	}
	acop := &prototype.AccountCreateOperation{
		Fee:            prototype.NewCoin(1),
		Creator:        &prototype.AccountName{Value: constants.COSInitMiner},
		NewAccountName: &prototype.AccountName{Value: accountName},
		Owner:          pub,
	}

	return acop, nil
}

func Test_PushTrx(t *testing.T) {
	// set up controller
	db := startDB()
	defer func() {
		stopGenerateBlock()
		clearDB(db)
	}()
	c := startController(db)
	go startGenerateBlock(c)

	acop, err := makeCreateAccountOP(accountNameBob, pubKeyBob)
	if err != nil {
		t.Error("makeCreateAccountOP error:", err)
	}

	signedTrx, err := createSigTrx(c, constants.InitminerPrivKey, acop)
	if err != nil {
		t.Error("createSigTrx error:", err)
	}

	invoice := c.PushTrx(signedTrx)
	if invoice.Status != prototype.StatusSuccess {
		t.Error("PushTrx return status error:", invoice.Status," info:",invoice.ErrorInfo)
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
	defer func() {
		time.Sleep(time.Second)
		clearDB(db)
	}()
	c := startController(db)

	createOP, err := makeCreateAccountOP(accountNameBob, pubKeyBob)
	if err != nil {
		t.Error("makeCreateAccountOP error:", err)
	}

	signedTrx, err := createSigTrx(c, constants.InitminerPrivKey, createOP)
	if err != nil {
		t.Error("createSigTrx error:", err)
	}

	sigBlk := makeBlockWithCommonTrx(c.GetProps().GetHeadBlockId(), 10, signedTrx)

	fmt.Println("block size:", proto.Size(sigBlk))

	c.PushBlock(sigBlk, prototype.Skip_nothing)

	bobName := &prototype.AccountName{Value: accountNameBob}
	bobWrap := table.NewSoAccountWrap(db, bobName)
	if !bobWrap.CheckExist() {
		t.Error("create account failed")
	}
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
		tInfo.CreatedTime = &prototype.TimePointSec{UtcSeconds: 0}
		tInfo.SigningKey = &prototype.PublicKeyType{Data: []byte{1}}
	}), "Witness Create Error")

	name2 := &prototype.AccountName{Value: "wit2"}
	witnessWrap2 := table.NewSoWitnessWrap(db, name2)
	mustNoError(witnessWrap2.Create(func(tInfo *table.SoWitness) {
		tInfo.Owner = name2
		tInfo.CreatedTime = &prototype.TimePointSec{UtcSeconds: 0}
		tInfo.SigningKey = &prototype.PublicKeyType{Data: []byte{2}}
	}), "Witness Create Error")

	witnesses, _ := c.GetWitnessTopN(10)

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

	signedTrx, err := createSigTrx(c, constants.InitminerPrivKey, createOP)
	if err != nil {
		t.Error("createSigTrx error:", err)
	}

	block := makeBlockWithCommonTrx(c.GetProps().GetHeadBlockId(), 6, signedTrx)

	fmt.Println("block size:", proto.Size(block))

	c.PushBlock(block, prototype.Skip_nothing)

	// second block
	createOP2, err := makeCreateAccountOP(accountNameTom, pubKeyTom)
	if err != nil {
		t.Error("makeCreateAccountOP error:", err)
	}

	signedTrx2, err := createSigTrx(c, constants.InitminerPrivKey, createOP2)
	if err != nil {
		t.Error("createSigTrx error:", err)
	}

	block2 := makeBlockWithCommonTrx(c.GetProps().GetHeadBlockId(), 9, signedTrx2)

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

	signedTrx, err := createSigTrx(c, constants.InitminerPrivKey, createOP)
	if err != nil {
		t.Error("createSigTrx error:", err)
	}

	block := makeBlockWithCommonTrx(c.GetProps().GetHeadBlockId(), 6, signedTrx)

	fmt.Println("block size:", proto.Size(block))

	c.PushBlock(block, prototype.Skip_nothing)

	// second block
	createOP2, err := makeCreateAccountOP(accountNameTom, pubKeyTom)
	if err != nil {
		t.Error("makeCreateAccountOP error:", err)
	}

	signedTrx2, err := createSigTrx(c, constants.InitminerPrivKey, createOP2)
	if err != nil {
		t.Error("createSigTrx error:", err)
	}

	block2 := makeBlockWithCommonTrx(c.GetProps().GetHeadBlockId(), 9, signedTrx2)

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

	err = c.PopBlock(1)
	if err == nil {
		t.Error("pop a irreversible block but no error")
	}
}

func Test_MixOp(t *testing.T) {

	db := startDB()
	defer clearDB(db)
	defer stopGenerateBlock()
	c := startController(db)
	go startGenerateBlock(c)

	// deploy contract
	data, _ := ioutil.ReadFile("./test_data/hello.wasm")
	abi, _ := ioutil.ReadFile("./test_data/hello.abi")
	compressedCode, _ := common.Compress(data)
	compressedAbi, _ := common.Compress(abi)
	deployOp := &prototype.ContractDeployOperation{
		Owner:    &prototype.AccountName{Value: "initminer"},
		Contract: "hello",
		Abi:      compressedAbi,
		Code:     compressedCode,
	}

	signedTrx, err := createSigTrx(c, constants.InitminerPrivKey, deployOp)
	if err != nil {
		t.Error("createSigTrx error:", err)
	}

	invoice := c.PushTrx(signedTrx)
	if invoice.Status != prototype.StatusSuccess {
		t.Error("PushTrx return status error:", invoice.Status)
	}

	// first op : call contract
	applyOp := &prototype.ContractApplyOperation{
		Caller:   &prototype.AccountName{Value: constants.COSInitMiner},
		Owner:    &prototype.AccountName{Value: constants.COSInitMiner},
		Contract: "hello",
		Method:   "hi",
		Params:   "[\"contentos\"]",
		//Amount:   &prototype.Coin{Value: 1000},
	}

	//
	miner := &prototype.AccountName{Value: constants.COSInitMiner}
	minerWrap := table.NewSoAccountWrap(db, miner)
	b := minerWrap.GetStamina()
	t.Log("before call initminer stamina:", b)
	//

	oldBalance := minerWrap.GetBalance()

	transferOp := &prototype.TransferOperation{
		From:   prototype.NewAccountName(constants.COSInitMiner),
		To:     prototype.NewAccountName("someone"),
		Amount: prototype.NewCoin(1),
	}

	signedTrx2, err := createSigTrx(c, constants.InitminerPrivKey, applyOp, transferOp)
	if err != nil {
		t.Error("createSigTrx error:", err)
	}

	invoice2 := c.PushTrx(signedTrx2)
	if invoice2.Status != prototype.StatusDeductGas {
		t.Error("PushTrx return status error:", invoice2)
	}
	fmt.Println(invoice2)

	//
	minerWrap2 := table.NewSoAccountWrap(db, miner)
	b2 := minerWrap2.GetStamina()
	t.Log("after call initminer stamina:", b2)
	//

	// right result:
	// 1. gas should be deduct
	// 2. transfer should be revert
	if b >= b2 {
		t.Error("gas error or db error")
	}
	newBalance := minerWrap.GetBalance()
	if newBalance.Value != oldBalance.Value {
		t.Error("db not revert")
	}
}

func Test_Stake_UnStake(t *testing.T) {
	db := startDB()
	defer func() {
		time.Sleep(time.Second)
		stopGenerateBlock()
		clearDB(db)
	}()
	c := startController(db)
	go startGenerateBlock(c)

	wraper := table.NewSoAccountWrap(db, prototype.NewAccountName(constants.COSInitMiner))
	if wraper.GetStakeVesting().Value != 0 {
		t.Error("stake vesting error")
	}

	stakeOp := &prototype.StakeOperation{
		Account: prototype.NewAccountName(constants.COSInitMiner),
		Amount:  prototype.NewCoin(100),
	}

	signedTrx, err := createSigTrx(c, constants.InitminerPrivKey, stakeOp)
	if err != nil {
		t.Error("createSigTrx error:", err)
	}

	invoice := c.PushTrx(signedTrx)
	if invoice.Status != prototype.StatusSuccess {
		t.Error("PushTrx return status error:", invoice.Status)
	}

	if wraper.GetStakeVesting().Value != 100 {
		t.Error("stake vesting error")
	}

	// un stake
	unStakeOp := &prototype.UnStakeOperation{
		Account: prototype.NewAccountName(constants.COSInitMiner),
		Amount:  prototype.NewCoin(100),
	}

	signedTrx2, err := createSigTrx(c, constants.InitminerPrivKey, unStakeOp)
	if err != nil {
		t.Error("createSigTrx error:", err)
	}

	// trick time, made a time valid trx first, then mod global time to let freeze time over
	addGlobalTime(db, constants.StakeFreezeTime+1)

	invoice2 := c.PushTrx(signedTrx2)
	if invoice2.Status != prototype.StatusSuccess {
		t.Error("PushTrx return status error:", invoice2.Status)
	}

	if wraper.GetStakeVesting().Value != 0 {
		t.Error("stake vesting error")
	}

	// stake wrong amount
	stakeOp2 := &prototype.StakeOperation{
		Account: prototype.NewAccountName(constants.COSInitMiner),
		Amount:  prototype.NewCoin(constants.COSInitSupply + 1),
	}

	signedTrx3, err := createSigTrx(c, constants.InitminerPrivKey, stakeOp2)
	if err != nil {
		t.Error("createSigTrx error:", err)
	}

	invoice3 := c.PushTrx(signedTrx3)
	if invoice3.Status != prototype.StatusDeductGas {
		t.Error("PushTrx return status error:", invoice3.Status)
	}

	// un stake wrong amount
	unStakeOp2 := &prototype.UnStakeOperation{
		Account: prototype.NewAccountName(constants.COSInitMiner),
		Amount:  prototype.NewCoin(1),
	}

	signedTrx4, err := createSigTrx(c, constants.InitminerPrivKey, unStakeOp2)
	if err != nil {
		t.Error("createSigTrx error:", err)
	}

	invoice4 := c.PushTrx(signedTrx4)
	if invoice4.Status == prototype.StatusSuccess {
		t.Error("PushTrx return status error:", invoice4.Status)
	}
}

func Test_StakeFreezeTime(t *testing.T) {
	db := startDB()
	defer func() {
		stopGenerateBlock()
		clearDB(db)
	}()
	c := startController(db)
	go startGenerateBlock(c)

	wraper := table.NewSoAccountWrap(db, prototype.NewAccountName(constants.COSInitMiner))
	if wraper.GetStakeVesting().Value != 0 {
		t.Error("stake vesting error")
	}

	stake(c, constants.COSInitMiner, constants.InitminerPrivKey, 100000)
	stakeTime := wraper.GetLastStakeTime()
	fmt.Println(stakeTime.UtcSeconds)

	// unstake immediately should error
	unStakeOp := &prototype.UnStakeOperation{
		Account: prototype.NewAccountName(constants.COSInitMiner),
		Amount:  prototype.NewCoin(1),
	}
	signedTrx, err := createSigTrx(c, constants.InitminerPrivKey, unStakeOp)
	if err != nil {
		t.Error("createSigTrx error:", err)
	}

	invoice := c.PushTrx(signedTrx)
	if invoice.Status == prototype.StatusSuccess {
		t.Error("PushTrx return status error:", invoice.Status)
	}

	// unstake after freeze

	unStakeOp2 := &prototype.UnStakeOperation{
		Account: prototype.NewAccountName(constants.COSInitMiner),
		Amount:  prototype.NewCoin(2),
	}
	signedTrx2, err := createSigTrx(c, constants.InitminerPrivKey, unStakeOp2)

	// trick time, made a time valid trx first, then mod global time to let freeze time over
	addGlobalTime(db, constants.StakeFreezeTime+1)

	if err != nil {
		t.Error("createSigTrx error:", err)
	}
	invoice2 := c.PushTrx(signedTrx2)
	if invoice2.Status != prototype.StatusSuccess {
		t.Error("PushTrx return status error:", invoice2.Status)
	}
}

func Test_Consume1(t *testing.T) {
	// set up controller
	db := startDB()
	defer func() {
		stopGenerateBlock()
		clearDB(db)
	}()
	c := startController(db)
	go startGenerateBlock(c)

	var value uint64 = 10000000
	ok := create_and_transfer(c, accountNameBob, pubKeyBob, value)
	if !ok {
		t.Error("create_and_transfer error")
		return
	}

	// stake
	if ok := stake(c, accountNameBob, priKeyBob, value); !ok {
		t.Error("stake error")
		return
	}

	bobName := &prototype.AccountName{Value: accountNameBob}
	bobWrap := table.NewSoAccountWrap(db, bobName)
	if bobWrap.GetStakeVesting().Value != value {
		t.Error("stake error")
	}

	// unstake need a pool's api to check remain stamina
	unStakeOp := &prototype.UnStakeOperation{
		Account: prototype.NewAccountName(accountNameBob),
		Amount:  prototype.NewCoin(value),
	}

	signedTrx3, err := createSigTrx(c, priKeyBob, unStakeOp)
	if err != nil {
		t.Error("createSigTrx error:", err)
	}

	// trick time, made a time valid trx first, then mod global time to let freeze time over
	addGlobalTime(db, constants.StakeFreezeTime+1)

	invoice3 := c.PushTrx(signedTrx3)
	if invoice3.Status != prototype.StatusSuccess {
		t.Error("PushTrx return status error:", invoice3.Status)
	}
	/*if c.GetRemainStamina(accountNameBob) != c.GetStaminaMax(accountNameBob) {
		t.Error("unstake error")
	}*/
}

func Test_Recover1(t *testing.T) {
	// set up controller
	db := startDB()
	defer func() {
		stopGenerateBlock()
		clearDB(db)
	}()
	c := startController(db)
	go startGenerateBlock(c)

	var value uint64 = 10000000
	ok := create_and_transfer(c, accountNameBob, pubKeyBob, value)
	if !ok {
		t.Error("create_and_transfer error")
		return
	}

	if ok := stake(c, accountNameBob, priKeyBob, value-1); !ok {
		t.Error("stake error")
		return
	}

	bobName := &prototype.AccountName{Value: accountNameBob}
	bobWrap := table.NewSoAccountWrap(db, bobName)
	if bobWrap.GetStakeVesting().Value != value-1 {
		t.Error("stake error")
	}
	useStamina := bobWrap.GetStaminaFree() + bobWrap.GetStamina()
	if useStamina == 0 {
		t.Error("stamina error")
	}

	// recover
	addGlobalHeadNumer(db, constants.WindowSize)

	transOp2 := &prototype.TransferOperation{
		From:   &prototype.AccountName{Value: accountNameBob},
		To:     &prototype.AccountName{Value: constants.COSInitMiner},
		Amount: prototype.NewCoin(1),
	}

	signedTrx3, err := createSigTrx(c, priKeyBob, transOp2)
	if err != nil {
		t.Error("createSigTrx error:", err)
	}
	netSize := proto.Size(signedTrx3)

	invoice3 := c.PushTrx(signedTrx3)
	if invoice3.Status != prototype.StatusSuccess {
		t.Error("PushTrx return status error:", invoice3.Status)
	}
	all := bobWrap.GetStamina() + bobWrap.GetStaminaFree()
	commonCpuGas := constants.CommonOpGas / constants.CpuConsumePointDen
	if all != uint64(commonCpuGas)+uint64(netSize*int(float64(constants.NetConsumePointNum)/float64(constants.NetConsumePointDen))) {
		t.Error("recover or consume error")
	}
}

func Test_Consume2(t *testing.T) {
	// set up controller
	db := startDB()
	defer func() {
		stopGenerateBlock()
		clearDB(db)
	}()
	c := startController(db)
	go startGenerateBlock(c)

	var value uint64 = 1000000
	if ok := create_and_transfer(c, accountNameBob, pubKeyBob, value); !ok {
		t.Error("create_and_transfer error")
		return
	}
	if ok := create_and_transfer(c, accountNameTom, pubKeyTom, value); !ok {
		t.Error("create_and_transfer error")
		return
	}
	// stake same
	if ok := stake(c, accountNameBob, priKeyBob, value-2); !ok {
		t.Error("stake error")
		return
	}
	if ok := stake(c, accountNameTom, priKeyTom, value-2); !ok {
		t.Error("stake error")
		return
	}
	// bob transfer twice, tom transfer once
	if ok := transfer(c, accountNameBob, constants.COSInitMiner, priKeyBob, 1); !ok {
		t.Error("transfer error")
		return
	}
	if ok := transfer(c, accountNameBob, accountNameTom, priKeyBob, 1); !ok {
		t.Error("transfer error")
		return
	}
	if ok := transfer(c, accountNameTom, constants.COSInitMiner, priKeyTom, 1); !ok {
		t.Error("transfer error")
		return
	}
	// check
	bobName := &prototype.AccountName{Value: accountNameBob}
	bobWrap := table.NewSoAccountWrap(db, bobName)
	bobUse := bobWrap.GetStaminaFree() + bobWrap.GetStamina()

	tomName := &prototype.AccountName{Value: accountNameTom}
	tomWrap := table.NewSoAccountWrap(db, tomName)
	tomUse := tomWrap.GetStaminaFree() + tomWrap.GetStamina()

	fmt.Println("b:", bobUse, " t:", tomUse)
	if bobUse <= tomUse {
		t.Error("stamina error")
	}
}

func transfer(c *TrxPool, from string, to string, fromPrikey string, value uint64) bool {

	transOp := &prototype.TransferOperation{
		From:   &prototype.AccountName{Value: from},
		To:     &prototype.AccountName{Value: to},
		Amount: prototype.NewCoin(value),
	}

	signedTrx, err := createSigTrx(c, fromPrikey, transOp)
	if err != nil {
		return false
	}

	invoice := c.PushTrx(signedTrx)
	if invoice.Status != prototype.StatusSuccess {
		return false
	}
	return true
}

func create_and_transfer(c *TrxPool, name string, pubkey string, value uint64) bool {
	acop, err := makeCreateAccountOP(name, pubkey)
	if err != nil {
		return false
	}

	transOp := &prototype.TransferOperation{
		From:   &prototype.AccountName{Value: constants.COSInitMiner},
		To:     &prototype.AccountName{Value: name},
		Amount: prototype.NewCoin(value),
	}

	signedTrx, err := createSigTrx(c, constants.InitminerPrivKey, acop, transOp)
	if err != nil {
		return false
	}

	invoice := c.PushTrx(signedTrx)
	if invoice.Status != prototype.StatusSuccess {
		return false
	}
	return true
}

func stake(c *TrxPool, name string, prikey string, value uint64) bool {
	stakeOp := &prototype.StakeOperation{
		Account: prototype.NewAccountName(name),
		Amount: prototype.NewCoin(value),
	}

	signedTrx2, err := createSigTrx(c, prikey, stakeOp)
	if err != nil {
		return false
	}

	invoice2 := c.PushTrx(signedTrx2)
	if invoice2.Status != prototype.StatusSuccess {
		return false
	}
	return true
}

func Test_TrxSize(t *testing.T) {
	db := startDB()
	defer func() {
		stopGenerateBlock()
		clearDB(db)
	}()
	c := startController(db)
	go startGenerateBlock(c)

	createOP, err := makeCreateAccountOP(accountNameBob, pubKeyBob)
	if err != nil {
		t.Error("makeCreateAccountOP error:", err)
	}
	trx1, _ := createSigTrx(c, constants.InitminerPrivKey, createOP)
	fmt.Println(proto.Size(trx1))

	top := &prototype.TransferOperation{
		From:   prototype.NewAccountName("aaa"),
		To:     prototype.NewAccountName("bbb"),
		Amount: prototype.NewCoin(100),
		Memo:   "hello this is a transfer",
	}
	trx2, _ := createSigTrx(c, constants.InitminerPrivKey, top)
	fmt.Println(proto.Size(trx2))

	pub, _ := prototype.PublicKeyFromWIF(pubKeyBob)
	cp := &prototype.ChainProperties{AccountCreationFee: prototype.NewCoin(100), MaximumBlockSize: 100}
	bpRegistOp := &prototype.BpRegisterOperation{
		Owner:           prototype.NewAccountName("aaa"),
		Url:             "www.google.com",
		BlockSigningKey: pub,
		Props:           cp,
	}
	trx3, _ := createSigTrx(c, constants.InitminerPrivKey, bpRegistOp)
	fmt.Println(proto.Size(trx3))

	bpUnReOp := &prototype.BpUnregisterOperation{Owner: prototype.NewAccountName("aaa")}
	trx4, _ := createSigTrx(c, constants.InitminerPrivKey, bpUnReOp)
	fmt.Println(proto.Size(trx4))

	bpV := &prototype.BpVoteOperation{
		Voter:   prototype.NewAccountName("aaa"),
		Witness: prototype.NewAccountName("bbb"),
	}
	trx5, _ := createSigTrx(c, constants.InitminerPrivKey, bpV)
	fmt.Println(proto.Size(trx5))

	// 1kb
	post := &prototype.PostOperation{Uuid: 1, Owner: prototype.NewAccountName("aaa"), Title: "aaa", Content: "asaasdadsaddsasdad" +
		"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda" +
		"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda" +
		"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda" +
		"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda" +
		"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda" +
		"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda" +
		"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda" +
		"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda",
	}
	trx6, _ := createSigTrx(c, constants.InitminerPrivKey, post)
	fmt.Println(proto.Size(trx6))

	reply := &prototype.ReplyOperation{Uuid: 1, Owner: prototype.NewAccountName("aaa"), Content: "asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda",
		ParentUuid: 1}
	trx7, _ := createSigTrx(c, constants.InitminerPrivKey, reply)
	fmt.Println(proto.Size(trx7))

	follow := &prototype.FollowOperation{Account: prototype.NewAccountName("aaa"), FAccount: prototype.NewAccountName("bbb")}
	trx8, _ := createSigTrx(c, constants.InitminerPrivKey, follow)
	fmt.Println(proto.Size(trx8))

	vote := &prototype.VoteOperation{Voter: prototype.NewAccountName("aaa"), Idx: 1}
	trx9, _ := createSigTrx(c, constants.InitminerPrivKey, vote)
	fmt.Println(proto.Size(trx9))

	tranToV := &prototype.TransferToVestingOperation{From: prototype.NewAccountName("aaa"), To: prototype.NewAccountName("bbb"), Amount: prototype.NewCoin(100)}
	trx10, _ := createSigTrx(c, constants.InitminerPrivKey, tranToV)
	fmt.Println(proto.Size(trx10))

	claim := &prototype.ClaimOperation{Account: prototype.NewAccountName("aaa"), Amount: 1}
	trx11, _ := createSigTrx(c, constants.InitminerPrivKey, claim)
	fmt.Println(proto.Size(trx11))

	claimAll := &prototype.ClaimAllOperation{Account: prototype.NewAccountName("aaa")}
	trx12, _ := createSigTrx(c, constants.InitminerPrivKey, claimAll)
	fmt.Println(proto.Size(trx12))

	//
	data, _ := ioutil.ReadFile("./test_data/hello.wasm")
	abi, _ := ioutil.ReadFile("./test_data/hello.abi")
	compressedCode, _ := common.Compress(data)
	compressedAbi, _ := common.Compress(abi)
	deployOp := &prototype.ContractDeployOperation{
		Owner:    &prototype.AccountName{Value: "initminer"},
		Contract: "hello",
		Abi:      compressedAbi,
		Code:     compressedCode,
	}

	trx13, _ := createSigTrx(c, constants.InitminerPrivKey, deployOp)
	fmt.Println(proto.Size(trx13))

	//
	applyOp := &prototype.ContractApplyOperation{
		Caller:   &prototype.AccountName{Value: "initminer"},
		Owner:    &prototype.AccountName{Value: "initminer"},
		Contract: "hello",
		Method:   "hi",
		Params:   "[\"contentos\"]",
		//Amount:   &prototype.Coin{Value: 1000},
	}
	trx14, _ := createSigTrx(c, constants.InitminerPrivKey, applyOp)
	fmt.Println(proto.Size(trx14))

	stake := &prototype.StakeOperation{Account: prototype.NewAccountName("aaa"), Amount: prototype.NewCoin(1)}
	trx16, _ := createSigTrx(c, constants.InitminerPrivKey, stake)
	fmt.Println(proto.Size(trx16))

	unStake := &prototype.UnStakeOperation{Account: prototype.NewAccountName("aaa"), Amount: prototype.NewCoin(1)}
	trx17, _ := createSigTrx(c, constants.InitminerPrivKey, unStake)
	fmt.Println(proto.Size(trx17))
}

func Test_Gas(t *testing.T) {

	db := startDB()
	defer func() {
		stopGenerateBlock()
		clearDB(db)
	}()
	c := startController(db)
	go startGenerateBlock(c)

	//
	miner := &prototype.AccountName{Value: "initminer"}
	minerWrap := table.NewSoAccountWrap(db, miner)
	s0 := minerWrap.GetStamina() + minerWrap.GetStaminaFree()
	t.Log("before initminer stamina use:", s0)
	//

	// deploy contract
	data, _ := ioutil.ReadFile("./test_data/hello.wasm")
	abi, _ := ioutil.ReadFile("./test_data/hello.abi")
	compressedCode, _ := common.Compress(data)
	compressedAbi, _ := common.Compress(abi)
	deployOp := &prototype.ContractDeployOperation{
		Owner:    &prototype.AccountName{Value: "initminer"},
		Contract: "hello",
		Abi:      compressedAbi,
		Code:     compressedCode,
	}

	signedTrx, err := createSigTrx(c, constants.InitminerPrivKey, deployOp)
	if err != nil {
		t.Error("createSigTrx error:", err)
	}

	invoice := c.PushTrx(signedTrx)
	if invoice.Status != prototype.StatusSuccess {
		t.Error("PushTrx return status error:", invoice.Status)
	}

	//pri, err := prototype.PrivateKeyFromWIF(constants.InitminerPrivKey)
	//if err != nil {
//		t.Error("PrivateKeyFromWIF error")
//	}
	//pre := &prototype.Sha256{Hash: make([]byte, 32)}
	//headBlockTime := 18
	//block1, err := c.GenerateAndApplyBlock(constants.COSInitMiner, pre, uint32(headBlockTime), pri, 0)

	//
	s1 := minerWrap.GetStamina() + minerWrap.GetStaminaFree()
	t.Log("after deploy initminer stamina use:", s1)
	//

	// call contract
	applyOp := &prototype.ContractApplyOperation{
		Caller:   &prototype.AccountName{Value: constants.COSInitMiner},
		Owner:    &prototype.AccountName{Value: constants.COSInitMiner},
		Contract: "hello",
		Method:   "hi",
		Params:   "[\"contentos\"]",
		//Amount:   &prototype.Coin{Value: 1000},
	}

	// call contract repeated
	headBlockTime := 0
	for i := 0; i < 4; i++ {
		headBlockTime += 1
		signedTrx2, err := createSigTrxTmp(c, constants.InitminerPrivKey, uint32(headBlockTime), applyOp)
		if err != nil {
			t.Error("createSigTrx error:", err)
		}
		invoice2 := c.PushTrx(signedTrx2)
		if invoice2.Status != prototype.StatusSuccess && invoice2.Status != prototype.StatusDeductGas {
			t.Error("PushTrx return status error:", invoice2.Status," info:",invoice2.ErrorInfo)
		}
	}

	//headBlockTime = 21
	//id := block1.Id()
	//pre = &prototype.Sha256{Hash: id.Data[:]}
	//block2, err := c.GenerateAndApplyBlock(constants.COSInitMiner, pre, uint32(headBlockTime), pri, 0)
	fmt.Println()
	//fmt.Println("block size:", len(block2.Transactions))
	//
	s2 := minerWrap.GetStamina() + minerWrap.GetStaminaFree()
	t.Log("after call contract initminer stamina use:", s2)
	//
}

func Test_Transfer(t *testing.T) {

	db := startDB()
	defer func() {
		stopGenerateBlock()
		clearDB(db)
	}()
	c := startController(db)
	go startGenerateBlock(c)

	//
	miner := &prototype.AccountName{Value: "initminer"}
	minerWrap := table.NewSoAccountWrap(db, miner)
	s0 := minerWrap.GetStamina() + minerWrap.GetStaminaFree()
	t.Log("before initminer stamina use:", s0)
	//

	// deploy contract
	cop, err := makeCreateAccountOP(accountNameTom, pubKeyTom)

	signedTrx, err := createSigTrx(c, constants.InitminerPrivKey, cop)
	if err != nil {
		t.Error("createSigTrx error:", err)
	}

	invoice := c.PushTrx(signedTrx)
	if invoice.Status != prototype.StatusSuccess {
		t.Error("PushTrx return status error:", invoice.Status)
	}

	//
	s1 := minerWrap.GetStamina() + minerWrap.GetStaminaFree()
	t.Log("after deploy initminer stamina use:", s1)
	//

	// call contract
	from := prototype.NewAccountName(constants.COSInitMiner)
	to := prototype.NewAccountName(accountNameTom)
	applyOp := &prototype.TransferOperation{
		From:   from,
		To:     to,
		Amount: prototype.NewCoin(1),
		Memo: "asaasdadsaddsasdad" +
			"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda" +
			"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda" +
			"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda" +
			"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda" +
			"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda" +
			"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda" +
			"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda" +
			"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda" +
			"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda" +
			"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda" +
			"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda" +
			"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda" +
			"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda" +
			"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda" +
			"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda" +
			"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda" +
			"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda" +
			"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda" +
			"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda" +
			"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda" +
			"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda" +
			"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda" +
			"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda" +
			"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda" +
			"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda" +
			"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda" +
			"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda" +
			"asdasdadasdasdasdasdasdasdasdasdasdadadasdasdasdadasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdasdadasdadasdasdasdasda",
	}

	// call contract repeated
	for i := 0; i < 3; i++ { // 1800 is trx max expiration limit
		signedTrx2, err := createSigTrxTmp(c, constants.InitminerPrivKey, uint32(i+1), applyOp)
		if err != nil {
			t.Error("createSigTrx error:", err)
		}
		invoice2 := c.PushTrx(signedTrx2)
		if invoice2.Status != prototype.StatusSuccess && invoice2.Status != prototype.StatusDeductGas {
			t.Error("PushTrx return status error:", invoice2.Status," info:",invoice2.ErrorInfo)
		}
	}

	//
	s2 := minerWrap.GetStamina() + minerWrap.GetStaminaFree()
	t.Log("after call contract initminer stamina use:", s2)
	//
}
