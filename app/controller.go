package app

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/common/eventloop"
	"github.com/coschain/contentos-go/common/logging"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/prototype"
	"github.com/golang/protobuf/proto"
	"time"
)

var(
	SINGLE_ID int32 = 1
)

type Controller struct {
	iservices.IController
	// lock for db write
	// pending_trx_list
	// DB Manager
	ctx    *node.ServiceContext
	evLoop *eventloop.EventLoop

	db      iservices.IDatabaseService
	noticer EventBus.Bus
	skip    prototype.SkipFlag

	_pending_tx           []*prototype.TransactionWrapper
	_isProducing          bool
	_currentTrxId         *prototype.Sha256
	_current_op_in_trx    uint16
	_currentBlockNum      uint64
	_current_trx_in_block int16
	dgpo *prototype.DynamicProperties
	havePendingTransaction bool
	idToRev  map[[32]byte]uint64

	stack *sessionStack
}

func (c *Controller) getDb() (iservices.IDatabaseService, error) {
	s, err := c.ctx.Service(iservices.DB_SERVER_NAME)
	if err != nil {
		return nil, err
	}
	db := s.(iservices.IDatabaseService)
	return db, nil
}

// for easy test
func (c *Controller) SetDB(db iservices.IDatabaseService) {
	c.db = db
}

// service constructor
func NewController(ctx *node.ServiceContext) (*Controller, error) {
	return &Controller{ctx: ctx}, nil
}

func (c *Controller) Start(node *node.Node) error {
	db, err := c.getDb()
	if err != nil {
		return err
	}
	c.db = db
	c.evLoop = node.MainLoop
	c.noticer = node.EvBus

	c.Open()
	return nil
}

func (c *Controller) Open() {
	dgpWrap := table.NewSoGlobalWrap(c.db, &SINGLE_ID )
	if !dgpWrap.CheckExist() {

		mustNoError(c.db.DeleteAll(), "truncate database error")

		logging.CLog().Info("start initGenesis")
		c.initGenesis()
		logging.CLog().Info("finish initGenesis")
	}
	c.dgpo = dgpWrap.GetProps()
}

func (c *Controller) Stop() error {
	return nil
}

func (c *Controller) setProducing(b bool) {
	c._isProducing = b
}

func (c *Controller) PushTrx(trx *prototype.SignedTransaction) *prototype.TransactionInvoice {
	// this function may be cross routines ? use channel or lock ?
	oldSkip := c.skip
	defer func() {
		if err := recover(); err != nil {
			logging.CLog().Errorf("PushTrx Error: %v", err)
		}
		c.setProducing(false)
		c.skip = oldSkip
	}()

	// check maximum_block_size
	mustSuccess(proto.Size(trx) <= int(c.dgpo.MaximumBlockSize-256),"transaction is too large")

	c.setProducing(true)
	return c._pushTrx(trx)
}

func (c *Controller) _pushTrx(trx *prototype.SignedTransaction) *prototype.TransactionInvoice {
	defer func() {
		// undo sub session
		if err := recover(); err != nil {
			c.db.EndTransaction(false)
			panic(err)
		}
	}()
	// start a new undo session when first transaction come after push block
	if !c.havePendingTransaction {
		c.db.BeginTransaction()
		c.havePendingTransaction = true
	}

	trxWrp := &prototype.TransactionWrapper{}
	trxWrp.SigTrx = trx
	trxWrp.Invoice = &prototype.TransactionInvoice{}

	// start a sub undo session for applyTransaction
	c.db.BeginTransaction()

	c._applyTransaction(trxWrp)
	c._pending_tx = append(c._pending_tx, trxWrp)

	// commit sub session
	c.db.EndTransaction(true)

	// @ not use yet
	//c.notifyTrxPending(trx)
	return trxWrp.Invoice
}

func (c *Controller) PushBlock(blk *prototype.SignedBlock, skip prototype.SkipFlag) {
	oldFlag := c.skip
	defer func() {
		c.skip = oldFlag
	}()

	c.skip = skip

	tmpPending := c.ClearPending()

	defer func() {
		if err := recover(); err != nil {
			c.skip = oldFlag
			c.restorePending(tmpPending)
			logging.CLog().Debug("push block error : ",err)
			panic("PushBlock error")
		}
	}()

	defer func() {
		c.restorePending(tmpPending)
	}()

	c.applyBlock(blk,skip)
}

func (c *Controller) ClearPending() []*prototype.TransactionWrapper {
	// @
	mustSuccess(len(c._pending_tx) == 0 || c.havePendingTransaction,"can not clear pending")
	res := make([]*prototype.TransactionWrapper, len(c._pending_tx))
	copy(res, c._pending_tx)

	c._pending_tx = c._pending_tx[:0]

	if c.skip & prototype.Skip_apply_transaction == 0 {
		c.db.EndTransaction(false)
		c.havePendingTransaction = false
	}

	return res
}

func (c *Controller) restorePending(pending []*prototype.TransactionWrapper) {
	for _, tw := range pending {
		id, err := tw.SigTrx.Id()
		mustNoError(err,"get transaction id error")

		objWrap := table.NewSoTransactionObjectWrap(c.db, id)
		if !objWrap.CheckExist() {
			c._pushTrx(tw.SigTrx)
		}
	}
}

func emptyHeader(signHeader * prototype.SignedBlockHeader) {
	signHeader.Header =  new(prototype.BlockHeader)
	signHeader.Header.Previous = &prototype.Sha256{}
	signHeader.Header.Timestamp = &prototype.TimePointSec{}
	signHeader.Header.Witness = &prototype.AccountName{}
	signHeader.Header.TransactionMerkleRoot = &prototype.Sha256{}
	signHeader.WitnessSignature = &prototype.SignatureType{}
}

func (c *Controller) GenerateBlock(witness string, timestamp uint32,
	priKey *prototype.PrivateKeyType, skip prototype.SkipFlag) *prototype.SignedBlock {
	oldSkip := c.skip
	defer func() {
		c.skip = oldSkip
		if err := recover(); err != nil {
			c.db.EndTransaction(false)
			logging.CLog().Errorf("GenerateBlock Error: %v", err)
			panic(err)
		}
	}()

	c.skip = skip

	/*
	slotNum := c.GetIncrementSlotAtTime(&prototype.TimePointSec{UtcSeconds:timestamp})
	mustSuccess(slotNum > 0,"slot num must > 0")
	witnessName := c.GetScheduledWitness(slotNum)
	mustSuccess(witnessName.Value == witness,"not this witness")*/

	pubkey,err := priKey.PubKey()
	mustNoError(err,"get public key error")

	witnessWrap := table.NewSoWitnessWrap(c.db,&prototype.AccountName{Value:witness})
	mustSuccess(bytes.Equal(witnessWrap.GetSigningKey().Data[:],pubkey.Data[:]),"public key not equal")

	// @ signHeader size is zero, must have some content
	signHeader := &prototype.SignedBlockHeader{}
	emptyHeader(signHeader)
	maxBlockHeaderSize := proto.Size(signHeader) + 4

	dgpWrap := table.NewSoGlobalWrap(c.db,&SINGLE_ID)
	maxBlockSize := dgpWrap.GetProps().MaximumBlockSize
	var totalSize uint32 = uint32(maxBlockHeaderSize)

	signBlock := &prototype.SignedBlock{}
	signBlock.SignedHeader = &prototype.SignedBlockHeader{}
	signBlock.SignedHeader.Header = &prototype.BlockHeader{}
	c._current_trx_in_block = 0

	// undo all _pending_tx in DB
	if c.havePendingTransaction {
		c.db.EndTransaction(false)
	}
	c.db.BeginTransaction()
	c.havePendingTransaction = true

	var postponeTrx uint64 = 0
	for _,trxWraper := range c._pending_tx {
		if trxWraper.SigTrx.Trx.Expiration.UtcSeconds < timestamp {
			continue
		}
		var newTotalSize uint64 = uint64(totalSize) + uint64(proto.Size(trxWraper))
		if newTotalSize > uint64(maxBlockSize) {
			postponeTrx++
			continue
		}

		func () {
			defer func() {
				if err := recover(); err != nil{
					c.db.EndTransaction(false)
				}
			}()

			c.db.BeginTransaction()
			c._applyTransaction(trxWraper)
			c.db.EndTransaction(true)

			totalSize += uint32(proto.Size(trxWraper))
			signBlock.Transactions = append(signBlock.Transactions,trxWraper)
			c._current_trx_in_block++
		}()
	}
	if postponeTrx > 0 {
		logging.CLog().Warnf("postponed %d trx due to max block size",postponeTrx)
	}

	signBlock.SignedHeader.Header.Previous = c.headBlockID()
	signBlock.SignedHeader.Header.Timestamp = &prototype.TimePointSec{UtcSeconds:timestamp}
	id := signBlock.CalculateMerkleRoot()
	signBlock.SignedHeader.Header.TransactionMerkleRoot = &prototype.Sha256{Hash:id.Data[:]}
	signBlock.SignedHeader.Header.Witness = &prototype.AccountName{Value:witness}
	signBlock.SignedHeader.WitnessSignature = &prototype.SignatureType{}
	signBlock.SignedHeader.Sign(priKey)

	mustSuccess(proto.Size(signBlock) <= constants.MAX_BLOCK_SIZE,"block size too big")

	/*c.PushBlock(signBlock,c.skip | prototype.Skip_apply_transaction)

	if signBlock.SignedHeader.Number() == uint64(c.headBlockNum()) {
		//c.db.EndTransaction(true)
	} else {
		c.db.EndTransaction(false)
	}*/

	c.db.EndTransaction(false)

	return signBlock
}

func (c *Controller) notifyOpPreExecute(on *prototype.OperationNotification) {
	c.noticer.Publish(constants.NOTICE_OP_PRE, on)
}

func (c *Controller) notifyOpPostExecute(on *prototype.OperationNotification) {
	c.noticer.Publish(constants.NOTICE_OP_POST, on)
}

func (c *Controller) notifyTrxPreExecute(trx *prototype.SignedTransaction) {
	c.noticer.Publish(constants.NOTICE_TRX_PRE, trx)
}

func (c *Controller) notifyTrxPostExecute(trx *prototype.SignedTransaction) {
	c.noticer.Publish(constants.NOTICE_TRX_POST, trx)
}

func (c *Controller) notifyTrxPending(trx *prototype.SignedTransaction) {
	c.noticer.Publish(constants.NOTICE_TRX_PENDING, trx)
}

func (c *Controller) notifyBlockApply(block *prototype.SignedBlock) {
	c.noticer.Publish(constants.NOTICE_BLOCK_APPLY, block)
}

// calculate reward for creator and witness
func (c *Controller) processBlock() {
}
func (c *Controller) applyTransaction(trxWrp *prototype.TransactionWrapper) {
	c._applyTransaction(trxWrp)
	// @ not use yet
	//c.notifyTrxPostExecute(trxWrp.SigTrx)
}
func (c *Controller) _applyTransaction(trxWrp *prototype.TransactionWrapper) {
	defer func() {
		if err := recover(); err != nil {
			trxWrp.Invoice.Status = 500
			panic(fmt.Sprintf("applyTransaction failed : %v", err))
		} else {
			trxWrp.Invoice.Status = 200
			return
		}
	}()

	trx := trxWrp.SigTrx
	var err error
	c._currentTrxId, err = trx.Id()
	mustNoError(err,"get trx id failed")

	trx.Validate()

	// trx duplicate check
	transactionObjWrap := table.NewSoTransactionObjectWrap(c.db, c._currentTrxId)
	mustSuccess(!transactionObjWrap.CheckExist(),"Duplicate transaction check failed")

	if c.skip&prototype.Skip_transaction_signatures == 0 {
		postingGetter := func(name string) *prototype.Authority {
			account := &prototype.AccountName{Value: name}
			authWrap := table.NewSoAccountAuthorityObjectWrap(c.db, account)
			auth := authWrap.GetPosting()
			if auth == nil {
				panic("no posting auth")
			}
			return auth
		}
		activeGetter := func(name string) *prototype.Authority {
			account := &prototype.AccountName{Value: name}
			authWrap := table.NewSoAccountAuthorityObjectWrap(c.db, account)
			auth := authWrap.GetActive()
			if auth == nil {
				panic("no posting auth")
			}
			return auth
		}
		ownerGetter := func(name string) *prototype.Authority {
			account := &prototype.AccountName{Value: name}
			authWrap := table.NewSoAccountAuthorityObjectWrap(c.db, account)
			auth := authWrap.GetOwner()
			if auth == nil {
				panic("no posting auth")
			}
			return auth
		}

		tmpChainId := prototype.ChainId{Value: 0}
		trx.VerifyAuthority(tmpChainId, 2, postingGetter, activeGetter, ownerGetter)
		// @ check_admin
	}

	blockNum := c.dgpo.GetHeadBlockNumber()
	if blockNum > 0 {
		uniWrap := table.UniBlockSummaryObjectIdWrap{Dba:c.db}
		idWrap := uniWrap.UniQueryId(&trx.Trx.RefBlockNum)
		if !idWrap.CheckExist() {
			panic("no refBlockNum founded")
		} else {
			blockId := idWrap.GetBlockId()
			summaryId := binary.BigEndian.Uint32(blockId.Hash[8:12])
			mustSuccess(trx.Trx.RefBlockPrefix ==  summaryId,"transaction tapos failed")
		}
		// get head time
		mustSuccess(trx.Trx.Expiration.UtcSeconds <= uint32(time.Now().Second()+30),"transaction expiration too long")
		mustSuccess(uint32(time.Now().Second()) <= trx.Trx.Expiration.UtcSeconds,"transaction has expired")
	}

	// insert trx into DB unique table
	cErr := transactionObjWrap.Create(func(tInfo *table.SoTransactionObject) {
		tInfo.TrxId = c._currentTrxId
		tInfo.Expiration = &prototype.TimePointSec{UtcSeconds: 100}
	})
	mustNoError(cErr,"create transactionObject failed")

	// @ not use yet
	//c.notifyTrxPreExecute(trx)

	// process operation
	c._current_op_in_trx = 0
	for _, op := range trx.Trx.Operations {
		c.applyOperation(op)
		c._current_op_in_trx++
	}

	c._currentTrxId = &prototype.Sha256{}
}

func (c *Controller) applyOperation(op *prototype.Operation) {
	// @ not use yet
	n := &prototype.OperationNotification{Op: op}
	c.notifyOpPreExecute(n)

	eva := c.getEvaluator(op)
	eva.Apply()

	// @ not use yet
	c.notifyOpPostExecute(n)
}

func (c *Controller) getEvaluator(op *prototype.Operation) BaseEvaluator {
	ctx := &ApplyContext{db: c.db, control: c}
	switch op.Op.(type) {
	case *prototype.Operation_Op1:
		eva := &AccountCreateEvaluator{ctx: ctx, op: op.GetOp1()}
		return BaseEvaluator(eva)
	case *prototype.Operation_Op2:
		eva := &TransferEvaluator{ctx: ctx, op: op.GetOp2()}
		return BaseEvaluator(eva)
	case *prototype.Operation_Op3:
		eva := &BpRegisterEvaluator{ctx: ctx, op: op.GetOp3()}
		return BaseEvaluator(eva)
	case *prototype.Operation_Op4:
		eva := &BpUnregisterEvaluator{ctx: ctx, op: op.GetOp4()}
		return BaseEvaluator(eva)
	case *prototype.Operation_Op5:
		eva := &BpVoteEvaluator{ctx: ctx, op: op.GetOp5()}
		return BaseEvaluator(eva)
	case *prototype.Operation_Op6:
		eva := &PostEvaluator{ctx: ctx, op: op.GetOp6()}
		return BaseEvaluator(eva)
	case *prototype.Operation_Op7:
		eva := &ReplyEvaluator{ctx: ctx, op: op.GetOp7()}
		return BaseEvaluator(eva)
	case *prototype.Operation_Op8:
		eva := &FollowEvaluator{ctx: ctx, op: op.GetOp8()}
		return BaseEvaluator(eva)
	case *prototype.Operation_Op9:
		eva := &VoteEvaluator{ctx: ctx, op: op.GetOp9()}
		return BaseEvaluator(eva)
	case *prototype.Operation_Op10:
		eva := &TransferToVestingEvaluator{ctx: ctx, op: op.GetOp10()}
		return BaseEvaluator(eva)
	default:
		panic("no matchable evaluator")
	}
}

func (c *Controller) applyBlock(blk *prototype.SignedBlock,skip prototype.SkipFlag) {
	oldFlag := c.skip
	defer func() {
		c.skip = oldFlag
	}()

	c.skip = skip
	c._applyBlock(blk,skip)

	// @ tps update
}

func (c *Controller) _applyBlock(blk *prototype.SignedBlock,skip prototype.SkipFlag) {
	nextBlockNum := blk.Id().BlockNum()

	merkleRoot := blk.CalculateMerkleRoot()
	mustSuccess(bytes.Equal(merkleRoot.Data[:], blk.SignedHeader.Header.TransactionMerkleRoot.Hash),"Merkle check failed")

	// validate_block_header
	c.validateBlockHeader(blk)

	c._currentBlockNum = nextBlockNum
	c._current_trx_in_block = 0

	blockSize := proto.Size(blk)
	mustSuccess(uint32(blockSize) <= c.dgpo.GetMaximumBlockSize(),"Block size is too big")

	if uint32(blockSize) < constants.MIN_BLOCK_SIZE {
		// elog("Block size is too small")
	}

	w := blk.SignedHeader.Header.Witness
	c.dgpo.CurrentWitness = w

	// @ process extension

	// @ hardfork_state

	trxWrp := &prototype.TransactionWrapper{}
	trxWrp.Invoice = &prototype.TransactionInvoice{}

	if skip & prototype.Skip_apply_transaction == 0{

		for _, tw := range blk.Transactions {
			trxWrp.SigTrx = tw.SigTrx
			trxWrp.Invoice.Status = 200
			c.applyTransaction(trxWrp)
			mustSuccess(trxWrp.Invoice.Status == tw.Invoice.Status,"mismatched invoice")
			c._current_trx_in_block++
		}
	}

	c.updateGlobalDynamicData(blk)
	//c.updateSigningWitness(blk)
	// @ update_last_irreversible_block
	c.createBlockSummary(blk)
	c.clearExpiredTransactions()
	// @ ...

	// @ notify_applied_block
}

func (c *Controller) initGenesis() {

	// create initminer
	pubKey, _ := prototype.PublicKeyFromWIF(constants.INITMINER_PUBKEY)
	name := &prototype.AccountName{Value: constants.INIT_MINER_NAME}
	newAccountWrap := table.NewSoAccountWrap(c.db, name)
	mustNoError(newAccountWrap.Create(func(tInfo *table.SoAccount) {
		tInfo.Name = name
		tInfo.CreatedTime = &prototype.TimePointSec{UtcSeconds: 0}
		tInfo.Balance = prototype.NewCoin(constants.INIT_SUPPLY)
		tInfo.VestingShares = prototype.NewVest(0)
	}), "CreateAccount error")

	// create account authority
	authorityWrap := table.NewSoAccountAuthorityObjectWrap(c.db, name)
	ownerAuth := prototype.NewAuthorityFromPubKey(pubKey)

	mustNoError(authorityWrap.Create(func(tInfo *table.SoAccountAuthorityObject) {
		tInfo.Account = name
		tInfo.Posting = ownerAuth
		tInfo.Active = ownerAuth
		tInfo.Owner = ownerAuth
	}), "CreateAccountAuthorityObject error ")

	// create witness_object
	witnessWrap := table.NewSoWitnessWrap(c.db, name)
	mustNoError(witnessWrap.Create(func(tInfo *table.SoWitness) {
		tInfo.Owner = name
		tInfo.WitnessScheduleType = &prototype.WitnessScheduleType{Value: prototype.WitnessScheduleType_miner}
		tInfo.CreatedTime = &prototype.TimePointSec{UtcSeconds: 0}
		tInfo.SigningKey = pubKey
		tInfo.LastWork = &prototype.Sha256{Hash: []byte{0}}
	}), "Witness Create Error")

	// create dynamic global properties
	dgpWrap := table.NewSoGlobalWrap(c.db, &SINGLE_ID)
	mustNoError(dgpWrap.Create(func(tInfo *table.SoGlobal) {
		tInfo.Id = SINGLE_ID
		tInfo.Props = &prototype.DynamicProperties{}
		tInfo.Props.CurrentWitness = name
		tInfo.Props.Time = &prototype.TimePointSec{UtcSeconds: constants.GENESIS_TIME}
		tInfo.Props.HeadBlockId = &prototype.Sha256{Hash: make([]byte, 32)}
		// @ recent_slots_filled
		// @ participation_count
		tInfo.Props.CurrentSupply = prototype.NewCoin(constants.COS_INIT_SUPPLY)
		tInfo.Props.TotalCos = prototype.NewCoin(constants.COS_INIT_SUPPLY)
		tInfo.Props.MaximumBlockSize = constants.MAX_BLOCK_SIZE
		tInfo.Props.TotalVestingShares = prototype.NewVest(0)
	}), "CreateDynamicGlobalProperties error")

	// create block summary
	for i := uint32(0); i < 0x100; i++ {
		wrap := table.NewSoBlockSummaryObjectWrap(c.db, &i)
		mustNoError(wrap.Create(func(tInfo *table.SoBlockSummaryObject) {
			tInfo.Id = i
			tInfo.BlockId = &prototype.Sha256{Hash:make([]byte,32)}
		}), "CreateBlockSummaryObject error")
	}

	// create witness scheduler
	witnessScheduleWrap := table.NewSoWitnessScheduleObjectWrap(c.db, &SINGLE_ID)
	mustNoError(witnessScheduleWrap.Create(func(tInfo *table.SoWitnessScheduleObject) {
		tInfo.Id = SINGLE_ID
		tInfo.CurrentShuffledWitness = append(tInfo.CurrentShuffledWitness, constants.COS_INIT_MINER)
	}),"CreateWitnessScheduleObject error")
}

func (c *Controller) TransferToVest(value *prototype.Coin) {

	cos := c.dgpo.GetTotalCos()
	vest := c.dgpo.GetTotalVestingShares()
	addVest := value.ToVest()

	mustNoError(cos.Sub(value), "TotalCos overflow")
	c.dgpo.TotalCos = cos

	mustNoError(vest.Add(addVest), "TotalVestingShares overflow")
	c.dgpo.TotalVestingShares = vest

	c.updateGlobalDataToDB()
}

func (c *Controller) TransferFromVest(value *prototype.Vest) {

	cos := c.dgpo.GetTotalCos()
	vest := c.dgpo.GetTotalVestingShares()
	addCos := value.ToCoin()

	mustNoError(cos.Add(addCos), "TotalCos overflow")
	c.dgpo.TotalCos = cos

	mustNoError(vest.Sub(value), "TotalVestingShares overflow")
	c.dgpo.TotalVestingShares = vest
	
	// TODO if op execute failed ???? how to revert ??
	c.updateGlobalDataToDB()
}

func (c *Controller) validateBlockHeader(blk *prototype.SignedBlock) {
	headID := c.headBlockID()
	if !bytes.Equal(headID.Hash, blk.SignedHeader.Header.Previous.Hash) {
		panic("hash not equal")
	}
	headTime := c.headBlockTime()
	if headTime.UtcSeconds >= blk.SignedHeader.Header.Timestamp.UtcSeconds {
		panic("block time is invalid")
	}

	// witness sig check
	witnessName := blk.SignedHeader.Header.Witness
	witnessWrap := table.NewSoWitnessWrap(c.db, witnessName)
	pubKey := witnessWrap.GetSigningKey()
	res, err := blk.SignedHeader.ValidateSig(pubKey)
	if !res || err != nil {
		panic("ValidateSig error")
	}

	// witness schedule check
	/*
	nextSlot := c.GetIncrementSlotAtTime(blk.SignedHeader.Header.Timestamp)
	if nextSlot == 0 {
		panic("next slot should be greater than 0")
	}*/

	/*scheduledWitness := c.GetScheduledWitness(nextSlot)
	if witnessWrap.GetOwner().Value != scheduledWitness.Value {
		panic("Witness produced block at wrong time")
	}*/
}

func (c *Controller) headBlockID() *prototype.Sha256 {
	return c.dgpo.GetHeadBlockId()
}

func (c *Controller) HeadBlockTime() *prototype.TimePointSec {
	return c.headBlockTime()
}
func (c *Controller) headBlockTime() *prototype.TimePointSec {
	return c.dgpo.Time
}

func (c *Controller) headBlockNum() uint32 {
	return c.dgpo.HeadBlockNumber
}

func (c *Controller) GetSlotTime(slot uint32) *prototype.TimePointSec {
	if slot == 0 {
		return &prototype.TimePointSec{UtcSeconds: 0}
	}

	if c.headBlockNum() == 0 {
		genesisTime := c.headBlockTime()
		genesisTime.UtcSeconds += slot * constants.BLOCK_INTERVAL
		return genesisTime
	}

	headBlockAbsSlot := c.headBlockTime().UtcSeconds / constants.BLOCK_INTERVAL
	slotTime := &prototype.TimePointSec{UtcSeconds: headBlockAbsSlot * constants.BLOCK_INTERVAL}

	slotTime.UtcSeconds += slot * constants.BLOCK_INTERVAL
	return slotTime
}

func (c *Controller) GetIncrementSlotAtTime(t *prototype.TimePointSec) uint32 {
	/*nextBlockSlotTime := c.GetSlotTime(1)
	if t.UtcSeconds < nextBlockSlotTime.UtcSeconds {
		return 0
	}
	return (t.UtcSeconds-nextBlockSlotTime.UtcSeconds)/constants.BLOCK_INTERVAL + 1*/
	return 0
}

func (c *Controller) GetScheduledWitness(slot uint32) *prototype.AccountName {
	currentSlot := c.dgpo.GetCurrentAslot()
	currentSlot += slot

	wsoWrap := table.NewSoWitnessScheduleObjectWrap(c.db, &SINGLE_ID)
	witnesses := wsoWrap.GetCurrentShuffledWitness()
	witnessNum := uint32(len(witnesses))
	witnessName := witnesses[currentSlot%witnessNum]
	return &prototype.AccountName{Value:witnessName}
}

func (c *Controller) updateGlobalDataToDB() {
	dgpWrap := table.NewSoGlobalWrap(c.db, &SINGLE_ID)
	mustSuccess( dgpWrap.MdProps( c.dgpo ), "")
}

func (c *Controller) updateGlobalDynamicData(blk *prototype.SignedBlock) {
	var missedBlock uint32 = 0
	/*
	if false && c.headBlockTime().UtcSeconds != 0 {
		missedBlock = c.GetIncrementSlotAtTime(blk.SignedHeader.Header.Timestamp)
		mustSuccess(missedBlock != 0,"missedBlock error")
		missedBlock--
		for i:= uint32(0);i<missedBlock;i++{
			witnessMissedName := c.GetScheduledWitness(i+1)
			witnessWrap := table.NewSoWitnessWrap(c.db,witnessMissedName)
			if witnessWrap.GetOwner().Value != blk.SignedHeader.Header.Witness.Value {
				oldMissed := witnessWrap.GetTotalMissed()
				oldMissed++
				witnessWrap.MdTotalMissed(oldMissed)
				if c.headBlockNum() - witnessWrap.GetLastConfirmedBlockNum() > constants.BLOCKS_PER_DAY {
					emptyKey := &prototype.PublicKeyType{Data:[]byte{0}}
					witnessWrap.MdSigningKey(emptyKey)
					// @ push push_virtual_operation shutdown_witness_operation
				}
			}
		}
	}*/

	// @ calculate participation

	id         := blk.Id()
	blockID    := &prototype.Sha256{Hash:id.Data[:]}

	c.dgpo.HeadBlockNumber    = uint32(blk.Id().BlockNum())
	c.dgpo.HeadBlockId        = blockID
	c.dgpo.Time               = blk.SignedHeader.Header.Timestamp
	c.dgpo.CurrentAslot       = c.dgpo.CurrentAslot + missedBlock+1

	// this check is useful ?
	mustSuccess( c.dgpo.GetHeadBlockNumber() - c.dgpo.GetIrreversibleBlockNum() < constants.MAX_UNDO_HISTORY,"The database does not have enough undo history to support a blockchain with so many missed blocks." )
	c.updateGlobalDataToDB()
}

func (c *Controller) updateSigningWitness(blk *prototype.SignedBlock) {
	/*newAsLot := c.dgpo.GetCurrentAslot() + c.GetIncrementSlotAtTime(blk.SignedHeader.Header.Timestamp)

	name := blk.SignedHeader.Header.Witness
	witnessWrap := table.NewSoWitnessWrap(c.db, name)
	witnessWrap.MdLastConfirmedBlockNum(uint32(blk.Id().BlockNum()))
	witnessWrap.MdLastAslot(newAsLot)*/
}

func (c *Controller) createBlockSummary(blk *prototype.SignedBlock) {
	blockNum := blk.Id().BlockNum()
	blockNumSuffix := uint32(blockNum & 0xffff)

	blockSummaryWrap := table.NewSoBlockSummaryObjectWrap(c.db, &blockNumSuffix)
	mustSuccess(blockSummaryWrap.CheckExist(),"can not get block summary object")
	blockIDArray := blk.Id().Data
	blockID := &prototype.Sha256{Hash: blockIDArray[:]}
	mustSuccess(blockSummaryWrap.MdBlockId(blockID),"update block summary object error")
}

func (c *Controller) clearExpiredTransactions() {
	sortWrap := table.STransactionObjectExpirationWrap{Dba: c.db}
	itr := sortWrap.QueryListByOrder(nil, nil) // query all
	if itr != nil {
		for itr.Next() {
			headTime := c.headBlockTime().UtcSeconds
			subPtr := sortWrap.GetSubVal(itr)
			if subPtr == nil {
				break
			}
			if headTime > subPtr.UtcSeconds {
				// delete trx ...
				k := sortWrap.GetMainVal(itr)
				objWrap := table.NewSoTransactionObjectWrap(c.db, k)
				mustSuccess(objWrap.RemoveTransactionObject(),"RemoveTransactionObject error")
			}
		}
		sortWrap.DelIterater(itr)
	}
}

func (c *Controller) GetWitnessTopN(n uint32) []string {
	ret := []string{}
	revList := table.SWitnessVoteCountWrap{Dba:c.db}
	itr := revList.QueryListByRevOrder(nil,nil)
	if itr != nil {
		i:= uint32(0)
		for itr.Next() && i < n {
			mainPtr := revList.GetMainVal(itr)
			if mainPtr != nil {
				ret = append(ret,mainPtr.Value)
			} else {
				// panic() ?
				logging.CLog().Warnf("reverse get witness meet nil value")
			}
			i++
		}
		revList.DelIterater(itr)
	}
	return ret
}

func (c *Controller) SetShuffledWitness(names []string) {
	witnessScheduleWrap := table.NewSoWitnessScheduleObjectWrap(c.db,&SINGLE_ID)
	mustSuccess(witnessScheduleWrap.MdCurrentShuffledWitness(names),"SetWitness error")
}

func (c *Controller) GetShuffledWitness() []string {
	witnessScheduleWrap := table.NewSoWitnessScheduleObjectWrap(c.db,&SINGLE_ID)
	return witnessScheduleWrap.GetCurrentShuffledWitness()
}

type reversion struct {
	self *common.BlockID
	pre *common.BlockID
}

func (c *Controller) saveReversion(id *common.BlockID) {
	currentRev := c.db.GetRevision()
	c.idToRev[id.Data] = currentRev
}

func (c *Controller) getReversion(id *common.BlockID) uint64{
	rev,ok := c.idToRev[id.Data]
	mustSuccess(ok,"reversion not found")
	return rev
}

func (c *Controller) gotoReversion(id *common.BlockID) {
	rev := c.getReversion(id)
	mustNoError(c.db.RevertToRevision(rev),"RevertToRevision error")
}

func (c *Controller) PopBlock() {
	// undo pending trx
	if c.havePendingTransaction {
		c.db.EndTransaction(false)
		c.havePendingTransaction = false
	}
	c.db.EndTransaction(false)
}

func (c *Controller) RevertTo(id *common.BlockID) {
	c.gotoReversion(id)
}

type layer struct{
	currentLayer uint32
	PendingLayer uint32
}

type sessionStack struct {
	db iservices.IDatabaseService
	layerInfo layer
	havePendingLayer bool
}

func (ss *sessionStack) begin(isPendingLayer bool) {
	ss.db.BeginTransaction()
	ss.layerInfo.currentLayer++
	if isPendingLayer {
		ss.layerInfo.PendingLayer = ss.layerInfo.currentLayer
	}
}

func (ss *sessionStack) squash() {
	mustSuccess(ss.layerInfo.currentLayer > 0, "squash when layer <= 0")
	mustSuccess(ss.layerInfo.PendingLayer + 1 == ss.layerInfo.currentLayer,"squash to no pending layer")
	ss.db.EndTransaction(true)
	ss.layerInfo.currentLayer--
}

func (ss *sessionStack) undo(isPendingLayer bool) {
	mustSuccess(ss.layerInfo.currentLayer > 0, "undo when layer <= 0")
	ss.db.EndTransaction(false)
	ss.layerInfo.currentLayer--
}

func (ss *sessionStack) keep() {

}