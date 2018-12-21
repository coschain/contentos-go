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
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/prototype"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"strconv"
)

var (
	SingleId int32 = 1
)

type TrxPool struct {
	iservices.ITrxPool
	// lock for db write
	// pending_trx_list
	// DB Manager
	ctx    *node.ServiceContext
	evLoop *eventloop.EventLoop

	db      iservices.IDatabaseService
	log     iservices.ILog
	noticer EventBus.Bus
	skip    prototype.SkipFlag

	pendingTx              []*prototype.TransactionWrapper
	isProducing            bool
	currentTrxId           *prototype.Sha256
	currentOpInTrx         uint16
	currentBlockNum        uint64
	currentTrxInBlock      int16
	havePendingTransaction bool
	shuffle                common.ShuffleFunc
}

func (c *TrxPool) getDb() (iservices.IDatabaseService, error) {
	s, err := c.ctx.Service(iservices.DbServerName)
	if err != nil {
		return nil, err
	}
	db := s.(iservices.IDatabaseService)
	return db, nil
}

func (c *TrxPool) getLog() (iservices.ILog, error) {
	s, err := c.ctx.Service(iservices.LogServerName)
	if err != nil {
		return nil, err
	}
	log := s.(iservices.ILog)
	return log, nil
}

func (c *TrxPool) SetShuffle(s common.ShuffleFunc) {
	c.shuffle = s
}

// for easy test
func (c *TrxPool) SetDB(db iservices.IDatabaseService) {
	c.db = db
}

func (c *TrxPool) SetBus(bus EventBus.Bus) {
	c.noticer = bus
}

func (c *TrxPool) SetLog(log iservices.ILog) {
	c.log = log
}

// service constructor
func NewController(ctx *node.ServiceContext) (*TrxPool, error) {
	return &TrxPool{ctx: ctx}, nil
}

func (c *TrxPool) Start(node *node.Node) error {
	log, err := c.getLog()
	if err != nil {
		return err
	}
	c.log = log

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

func (c *TrxPool) Open() {
	dgpWrap := table.NewSoGlobalWrap(c.db, &SingleId)
	if !dgpWrap.CheckExist() {

		mustNoError(c.db.DeleteAll(), "truncate database error")

		//c.log.GetLog().Info("start initGenesis")
		c.initGenesis()
		c.saveReversion(0)
		//c.log.GetLog().Info("finish initGenesis")
	}
}

func (c *TrxPool) Stop() error {
	return nil
}

func (c *TrxPool) setProducing(b bool) {
	c.isProducing = b
}

func (c *TrxPool) PushTrxToPending(trx *prototype.SignedTransaction) {

	if !c.havePendingTransaction {
		c.db.BeginTransaction()
		c.havePendingTransaction = true
	}

	trxWrp := &prototype.TransactionWrapper{}
	trxWrp.SigTrx = trx
	trxWrp.Invoice = &prototype.TransactionInvoice{}

	c.pendingTx = append(c.pendingTx, trxWrp)
}

func (c *TrxPool) PushTrx(trx *prototype.SignedTransaction) (invoice *prototype.TransactionInvoice) {
	// this function may be cross routines ? use channel or lock ?
	oldSkip := c.skip
	defer func() {
		if err := recover(); err != nil {
			invoice = &prototype.TransactionInvoice{Status: uint32(500)}
			//c.log.GetLog().Errorf("PushTrx Error: %v", err)
		}
		c.setProducing(false)
		c.skip = oldSkip
	}()

	// check maximum_block_size
	mustSuccess(proto.Size(trx) <= int(c.GetProps().MaximumBlockSize-256), "transaction is too large")

	c.setProducing(true)
	return c.pushTrx(trx)
}

func (c *TrxPool) GetProps() *prototype.DynamicProperties {
	dgpWrap := table.NewSoGlobalWrap(c.db, &SingleId)
	return dgpWrap.GetProps()
}

func (c *TrxPool) pushTrx(trx *prototype.SignedTransaction) *prototype.TransactionInvoice {
	defer func() {
		// undo sub session
		if err := recover(); err != nil {
			mustNoError(c.db.EndTransaction(false), "EndTransaction error")
			panic(err)
		}
	}()
	// start a new undo session when first transaction come after push block
	if !c.havePendingTransaction {
		c.db.BeginTransaction()
		//	c.log.GetLog().Debug("@@@@@@ pushTrx havePendingTransaction=true")
		c.havePendingTransaction = true
	}

	trxWrp := &prototype.TransactionWrapper{}
	trxWrp.SigTrx = trx
	trxWrp.Invoice = &prototype.TransactionInvoice{}

	// start a sub undo session for applyTransaction
	c.db.BeginTransaction()

	c.applyTransactionInner(trxWrp)
	c.pendingTx = append(c.pendingTx, trxWrp)

	// commit sub session
	mustNoError(c.db.EndTransaction(true), "EndTransaction error")

	// @ not use yet
	//c.notifyTrxPending(trx)
	return trxWrp.Invoice
}

func (c *TrxPool) PushBlock(blk *prototype.SignedBlock, skip prototype.SkipFlag) error {
	var err error = nil
	oldFlag := c.skip
	c.skip = skip

	tmpPending := c.ClearPending()

	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {
			case error:
				err = x
				//c.log.GetLog().Errorf("push block error : %v", x.Error())
			case string:
				err = errors.New(x)
				//c.log.GetLog().Errorf("push block error : %v ", x)
			default:
				err = errors.New("unknown panic type")
			}
			// undo changes
			c.db.EndTransaction(false)
			if skip&prototype.Skip_apply_transaction != 0 {
				c.havePendingTransaction = false
			}
		}
		c.skip = oldFlag
		// restorePending will call pushTrx, will start new transaction for pending
		c.restorePending(tmpPending)
	}()

	if skip&prototype.Skip_apply_transaction == 0 {
		c.db.BeginTransaction()
		c.applyBlock(blk, skip)
		mustNoError(c.db.EndTransaction(true), "EndTransaction error")
	} else {
		// we have do a BeginTransaction at GenerateBlock
		c.applyBlock(blk, skip)
		mustNoError(c.db.EndTransaction(true), "EndTransaction error")
		c.havePendingTransaction = false
	}

	blockNum := blk.Id().BlockNum()
	c.saveReversion(blockNum)
	return err
}

func (c *TrxPool) ClearPending() []*prototype.TransactionWrapper {
	// @
	mustSuccess(len(c.pendingTx) == 0 || c.havePendingTransaction, "can not clear pending")
	res := make([]*prototype.TransactionWrapper, len(c.pendingTx))
	copy(res, c.pendingTx)

	c.pendingTx = c.pendingTx[:0]

	// 1. block from network, we undo pending state
	// 2. block from local generate, we keep it
	if c.skip&prototype.Skip_apply_transaction == 0 {
		if c.havePendingTransaction == true {
			mustNoError(c.db.EndTransaction(false), "EndTransaction error")
			c.havePendingTransaction = false
			//		c.log.GetLog().Debug("@@@@@@ ClearPending havePendingTransaction=false")
		}
	}

	return res
}

func (c *TrxPool) restorePending(pending []*prototype.TransactionWrapper) {
	for _, tw := range pending {
		id, err := tw.SigTrx.Id()
		mustNoError(err, "get transaction id error")

		objWrap := table.NewSoTransactionObjectWrap(c.db, id)
		if !objWrap.CheckExist() {
			c.pushTrx(tw.SigTrx)
		}
	}
}

func emptyHeader(signHeader *prototype.SignedBlockHeader) {
	signHeader.Header = new(prototype.BlockHeader)
	signHeader.Header.Previous = &prototype.Sha256{}
	signHeader.Header.Timestamp = &prototype.TimePointSec{}
	signHeader.Header.Witness = &prototype.AccountName{}
	signHeader.Header.TransactionMerkleRoot = &prototype.Sha256{}
	signHeader.WitnessSignature = &prototype.SignatureType{}
}

func (c *TrxPool) GenerateAndApplyBlock(witness string, pre *prototype.Sha256, timestamp uint32,
	priKey *prototype.PrivateKeyType, skip prototype.SkipFlag) (*prototype.SignedBlock, error) {

	newBlock := c.GenerateBlock(witness, pre, timestamp, priKey, skip)

	err := c.PushBlock(newBlock, c.skip|prototype.Skip_apply_transaction)
	if err != nil {
		return nil, err
	}

	return newBlock, nil
}

func (c *TrxPool) GenerateBlock(witness string, pre *prototype.Sha256, timestamp uint32,
	priKey *prototype.PrivateKeyType, skip prototype.SkipFlag) *prototype.SignedBlock {
	oldSkip := c.skip
	defer func() {
		c.skip = oldSkip
		if err := recover(); err != nil {
			//mustNoError(c.db.EndTransaction(false), "EndTransaction error")
			//c.log.GetLog().Errorf("GenerateBlock Error: %v", err)
			panic(err)
		}
	}()

	c.skip = skip

	/*
		slotNum := c.GetIncrementSlotAtTime(&prototype.TimePointSec{UtcSeconds:timestamp})
		mustSuccess(slotNum > 0,"slot num must > 0")
		witnessName := c.GetScheduledWitness(slotNum)
		mustSuccess(witnessName.Value == witness,"not this witness")*/

	pubkey, err := priKey.PubKey()
	mustNoError(err, "get public key error")

	witnessWrap := table.NewSoWitnessWrap(c.db, &prototype.AccountName{Value: witness})
	mustSuccess(bytes.Equal(witnessWrap.GetSigningKey().Data[:], pubkey.Data[:]), "public key not equal")

	// @ signHeader size is zero, must have some content
	signHeader := &prototype.SignedBlockHeader{}
	emptyHeader(signHeader)
	maxBlockHeaderSize := proto.Size(signHeader) + 4

	dgpWrap := table.NewSoGlobalWrap(c.db, &SingleId)
	maxBlockSize := dgpWrap.GetProps().MaximumBlockSize
	var totalSize uint32 = uint32(maxBlockHeaderSize)

	signBlock := &prototype.SignedBlock{}
	signBlock.SignedHeader = &prototype.SignedBlockHeader{}
	signBlock.SignedHeader.Header = &prototype.BlockHeader{}
	c.currentTrxInBlock = 0

	// undo all pending in DB
	if c.havePendingTransaction {
		mustNoError(c.db.EndTransaction(false), "EndTransaction error")
	}
	c.db.BeginTransaction()
	//c.log.GetLog().Debug("@@@@@@ GeneratBlock havePendingTransaction=true")
	c.havePendingTransaction = true

	var postponeTrx uint64 = 0
	for _, trxWraper := range c.pendingTx {
		if trxWraper.SigTrx.Trx.Expiration.UtcSeconds < timestamp {
			continue
		}
		var newTotalSize uint64 = uint64(totalSize) + uint64(proto.Size(trxWraper))
		if newTotalSize > uint64(maxBlockSize) {
			postponeTrx++
			continue
		}

		func() {
			defer func() {
				if err := recover(); err != nil {
					mustNoError(c.db.EndTransaction(false), "EndTransaction error")
				}
			}()

			c.db.BeginTransaction()
			c.applyTransactionInner(trxWraper)
			mustNoError(c.db.EndTransaction(true), "EndTransaction error")

			totalSize += uint32(proto.Size(trxWraper))
			signBlock.Transactions = append(signBlock.Transactions, trxWraper)
			c.currentTrxInBlock++
		}()
	}
	if postponeTrx > 0 {
		//c.log.GetLog().Warnf("postponed %d trx due to max block size", postponeTrx)
	}

	signBlock.SignedHeader.Header.Previous = pre
	signBlock.SignedHeader.Header.Timestamp = &prototype.TimePointSec{UtcSeconds: timestamp}
	id := signBlock.CalculateMerkleRoot()
	signBlock.SignedHeader.Header.TransactionMerkleRoot = &prototype.Sha256{Hash: id.Data[:]}
	signBlock.SignedHeader.Header.Witness = &prototype.AccountName{Value: witness}
	signBlock.SignedHeader.WitnessSignature = &prototype.SignatureType{}
	signBlock.SignedHeader.Sign(priKey)

	mustSuccess(proto.Size(signBlock) <= constants.MAX_BLOCK_SIZE, "block size too big")
	// clearpending then let dpos call PushBlock, the point is without restore pending step when PushBlock
	//c.ClearPending()

	/*mustNoError(c.db.EndTransaction(false), "EndTransaction error")
	c.havePendingTransaction = false*/
	//c.log.GetLog().Debug("@@@@@@ GenerateBlock havePendingTransaction=false")

	/*c.PushBlock(signBlock,c.skip | prototype.Skip_apply_transaction)

	if signBlock.SignedHeader.Number() == uint64(c.headBlockNum()) {
		c.db.EndTransaction(true)
		c.saveReversion(uint32(signBlock.Id().BlockNum()))
	} else {
		c.db.EndTransaction(false)
	}*/

	return signBlock
}

func (c *TrxPool) notifyOpPreExecute(on *prototype.OperationNotification) {
	c.noticer.Publish(constants.NOTICE_OP_PRE, on)
}

func (c *TrxPool) notifyOpPostExecute(on *prototype.OperationNotification) {
	c.noticer.Publish(constants.NOTICE_OP_POST, on)
}

func (c *TrxPool) notifyTrxPreExecute(trx *prototype.SignedTransaction) {
	c.noticer.Publish(constants.NOTICE_TRX_PRE, trx)
}

func (c *TrxPool) notifyTrxPostExecute(trx *prototype.SignedTransaction) {
	c.noticer.Publish(constants.NOTICE_TRX_POST, trx)
}

func (c *TrxPool) notifyTrxPending(trx *prototype.SignedTransaction) {
	c.noticer.Publish(constants.NOTICE_TRX_PENDING, trx)
}

func (c *TrxPool) notifyBlockApply(block *prototype.SignedBlock) {
	c.noticer.Publish(constants.NOTICE_BLOCK_APPLY, block)
}

func (c *TrxPool) applyTransaction(trxWrp *prototype.TransactionWrapper) {
	c.applyTransactionInner(trxWrp)
	// @ not use yet
	//c.notifyTrxPostExecute(trxWrp.SigTrx)
}

func (c *TrxPool) applyTransactionInner(trxWrp *prototype.TransactionWrapper) {
	trxContext := NewTrxContext(trxWrp, c.db)
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
	c.currentTrxId, err = trx.Id()
	mustNoError(err, "get trx id failed")

	trx.Validate()

	// trx duplicate check
	transactionObjWrap := table.NewSoTransactionObjectWrap(c.db, c.currentTrxId)
	mustSuccess(!transactionObjWrap.CheckExist(), "Duplicate transaction check failed")

	if c.skip&prototype.Skip_transaction_signatures == 0 {
		tmpChainId := prototype.ChainId{Value: 0}
		mustNoError(trxContext.InitSigState(tmpChainId), "signature export error")
		trxContext.VerifySignature()
		// @ check_admin
	}

	blockNum := c.GetProps().GetHeadBlockNumber()
	if blockNum > 0 {
		uniWrap := table.UniBlockSummaryObjectIdWrap{Dba: c.db}
		idWrap := uniWrap.UniQueryId(&trx.Trx.RefBlockNum)
		if !idWrap.CheckExist() {
			panic("no refBlockNum founded")
		} else {
			blockId := idWrap.GetBlockId()
			summaryId := binary.BigEndian.Uint32(blockId.Hash[8:12])
			mustSuccess(trx.Trx.RefBlockPrefix == summaryId, "transaction tapos failed")
		}

		now := c.GetProps().Time
		// get head time
		mustSuccess(trx.Trx.Expiration.UtcSeconds <= uint32(now.UtcSeconds+constants.TRX_MAX_EXPIRATION_TIME), "transaction expiration too long")
		mustSuccess(now.UtcSeconds < trx.Trx.Expiration.UtcSeconds, "transaction has expired")
	}

	// insert trx into DB unique table
	cErr := transactionObjWrap.Create(func(tInfo *table.SoTransactionObject) {
		tInfo.TrxId = c.currentTrxId
		tInfo.Expiration = trx.Trx.Expiration
	})
	mustNoError(cErr, "create transactionObject failed")

	// @ not use yet
	//c.notifyTrxPreExecute(trx)

	// process operation
	c.currentOpInTrx = 0
	for _, op := range trx.Trx.Operations {
		c.applyOperation(trxContext, op)
		c.currentOpInTrx++
	}

	c.currentTrxId = &prototype.Sha256{}
}

func (c *TrxPool) applyOperation(trxCtx *TrxContext, op *prototype.Operation) {
	// @ not use yet
	n := &prototype.OperationNotification{Op: op}
	c.notifyOpPreExecute(n)

	eva := c.getEvaluator(trxCtx, op)
	eva.Apply()

	// @ not use yet
	c.notifyOpPostExecute(n)
}

func (c *TrxPool) getEvaluator(trxCtx *TrxContext, op *prototype.Operation) BaseEvaluator {
	ctx := &ApplyContext{db: c.db, control: c, trxCtx: trxCtx}
	return GetBaseEvaluator(ctx, op)
}

func (c *TrxPool) applyBlock(blk *prototype.SignedBlock, skip prototype.SkipFlag) {
	oldFlag := c.skip
	defer func() {
		c.skip = oldFlag
	}()

	c.skip = skip
	c.applyBlockInner(blk, skip)

	// @ tps update
}

func (c *TrxPool) applyBlockInner(blk *prototype.SignedBlock, skip prototype.SkipFlag) {
	nextBlockNum := blk.Id().BlockNum()

	merkleRoot := blk.CalculateMerkleRoot()
	mustSuccess(bytes.Equal(merkleRoot.Data[:], blk.SignedHeader.Header.TransactionMerkleRoot.Hash), "Merkle check failed")

	// validate_block_header
	c.validateBlockHeader(blk)

	c.currentBlockNum = nextBlockNum
	c.currentTrxInBlock = 0

	blockSize := proto.Size(blk)
	mustSuccess(uint32(blockSize) <= c.GetProps().GetMaximumBlockSize(), "Block size is too big")

	if uint32(blockSize) < constants.MIN_BLOCK_SIZE {
		// elog("Block size is too small")
	}

	w := blk.SignedHeader.Header.Witness
	dgpo := c.GetProps()
	dgpo.CurrentWitness = w
	dgpWrap := table.NewSoGlobalWrap(c.db, &SingleId)
	dgpWrap.MdProps(dgpo)

	// @ process extension

	// @ hardfork_state

	trxWrp := &prototype.TransactionWrapper{}
	trxWrp.Invoice = &prototype.TransactionInvoice{}

	if skip&prototype.Skip_apply_transaction == 0 {

		for _, tw := range blk.Transactions {
			trxWrp.SigTrx = tw.SigTrx
			trxWrp.Invoice.Status = 200
			c.applyTransaction(trxWrp)
			mustSuccess(trxWrp.Invoice.Status == tw.Invoice.Status, "mismatched invoice")
			c.currentTrxInBlock++
		}
	}

	c.updateGlobalDynamicData(blk)
	//c.updateSigningWitness(blk)
	c.shuffle(blk)
	// @ update_last_irreversible_block
	c.createBlockSummary(blk)
	c.clearExpiredTransactions()
	// @ ...

	// @ notify_applied_block
}

func (c *TrxPool) initGenesis() {

	c.db.BeginTransaction()
	defer func() {
		if err := recover(); err != nil {
			mustNoError(c.db.EndTransaction(false), "EndTransaction error")
			panic(err)
		} else {
			mustNoError(c.db.EndTransaction(true), "EndTransaction error")
		}
	}()
	// create initminer
	pubKey, _ := prototype.PublicKeyFromWIF(constants.INITMINER_PUBKEY)
	name := &prototype.AccountName{Value: constants.INIT_MINER_NAME}
	newAccountWrap := table.NewSoAccountWrap(c.db, name)
	mustNoError(newAccountWrap.Create(func(tInfo *table.SoAccount) {
		tInfo.Name = name
		tInfo.CreatedTime = &prototype.TimePointSec{UtcSeconds: 0}
		tInfo.Balance = prototype.NewCoin(constants.INIT_SUPPLY - 1000)
		tInfo.VestingShares = prototype.NewVest(1000)
		tInfo.LastPostTime = &prototype.TimePointSec{UtcSeconds: 0}
		tInfo.LastVoteTime = &prototype.TimePointSec{UtcSeconds: 0}
	}), "CreateAccount error")

	// create account authority
	authorityWrap := table.NewSoAccountAuthorityObjectWrap(c.db, name)
	ownerAuth := prototype.NewAuthorityFromPubKey(pubKey)

	mustNoError(authorityWrap.Create(func(tInfo *table.SoAccountAuthorityObject) {
		tInfo.Account = name
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
	dgpWrap := table.NewSoGlobalWrap(c.db, &SingleId)
	mustNoError(dgpWrap.Create(func(tInfo *table.SoGlobal) {
		tInfo.Id = SingleId
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

	//create rewards keeper
	keeperWrap := table.NewSoRewardsKeeperWrap(c.db, &SingleId)
	rewards := make(map[string]*prototype.Vest)
	rewards["initminer"] = &prototype.Vest{Value: 0}
	mustNoError(keeperWrap.Create(func(tInfo *table.SoRewardsKeeper) {
		tInfo.Id = SingleId
		//tInfo.Keeper.Rewards = map[string]*prototype.Vest{}
		tInfo.Keeper = &prototype.InternalRewardsKeeper{Rewards: rewards}
	}), "Create Rewards Keeper error")

	// create block summary buffer 2048
	for i := uint32(0); i < 0x800; i++ {
		wrap := table.NewSoBlockSummaryObjectWrap(c.db, &i)
		mustNoError(wrap.Create(func(tInfo *table.SoBlockSummaryObject) {
			tInfo.Id = i
			tInfo.BlockId = &prototype.Sha256{Hash: make([]byte, 32)}
		}), "CreateBlockSummaryObject error")
	}

	// create witness scheduler
	witnessScheduleWrap := table.NewSoWitnessScheduleObjectWrap(c.db, &SingleId)
	mustNoError(witnessScheduleWrap.Create(func(tInfo *table.SoWitnessScheduleObject) {
		tInfo.Id = SingleId
		tInfo.CurrentShuffledWitness = append(tInfo.CurrentShuffledWitness, constants.COS_INIT_MINER)
	}), "CreateWitnessScheduleObject error")
}

func (c *TrxPool) TransferToVest(value *prototype.Coin) {

	dgpo := c.GetProps()
	cos := dgpo.GetTotalCos()
	vest := dgpo.GetTotalVestingShares()
	addVest := value.ToVest()

	mustNoError(cos.Sub(value), "TotalCos overflow")
	dgpo.TotalCos = cos

	mustNoError(vest.Add(addVest), "TotalVestingShares overflow")
	dgpo.TotalVestingShares = vest

	c.updateGlobalDataToDB(dgpo)
}

func (c *TrxPool) TransferFromVest(value *prototype.Vest) {
	dgpo := c.GetProps()

	cos := dgpo.GetTotalCos()
	vest := dgpo.GetTotalVestingShares()
	addCos := value.ToCoin()

	mustNoError(cos.Add(addCos), "TotalCos overflow")
	dgpo.TotalCos = cos

	mustNoError(vest.Sub(value), "TotalVestingShares overflow")
	dgpo.TotalVestingShares = vest

	// TODO if op execute failed ???? how to revert ??
	c.updateGlobalDataToDB(dgpo)
}

func (c *TrxPool) validateBlockHeader(blk *prototype.SignedBlock) {
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

func (c *TrxPool) headBlockID() *prototype.Sha256 {
	return c.GetProps().GetHeadBlockId()
}

func (c *TrxPool) HeadBlockTime() *prototype.TimePointSec {
	return c.headBlockTime()
}
func (c *TrxPool) headBlockTime() *prototype.TimePointSec {
	return c.GetProps().Time
}

func (c *TrxPool) headBlockNum() uint64 {
	return c.GetProps().HeadBlockNumber
}

func (c *TrxPool) GetSlotTime(slot uint32) *prototype.TimePointSec {
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

func (c *TrxPool) GetIncrementSlotAtTime(t *prototype.TimePointSec) uint32 {
	/*nextBlockSlotTime := c.GetSlotTime(1)
	if t.UtcSeconds < nextBlockSlotTime.UtcSeconds {
		return 0
	}
	return (t.UtcSeconds-nextBlockSlotTime.UtcSeconds)/constants.BLOCK_INTERVAL + 1*/
	return 0
}

func (c *TrxPool) GetScheduledWitness(slot uint32) *prototype.AccountName {
	return nil
	/*
		currentSlot := c.dgpo.GetCurrentAslot()
		currentSlot += slot

		wsoWrap := table.NewSoWitnessScheduleObjectWrap(c.db, &SingleId)
		witnesses := wsoWrap.GetCurrentShuffledWitness()
		witnessNum := uint32(len(witnesses))
		witnessName := witnesses[currentSlot%witnessNum]
		return &prototype.AccountName{Value:witnessName}*/
}

func (c *TrxPool) updateGlobalDataToDB(dgpo *prototype.DynamicProperties) {
	dgpWrap := table.NewSoGlobalWrap(c.db, &SingleId)
	mustSuccess(dgpWrap.MdProps(dgpo), "")
}

func (c *TrxPool) updateGlobalDynamicData(blk *prototype.SignedBlock) {
	/*var missedBlock uint32 = 0

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
		}*/

	// @ calculate participation

	id := blk.Id()
	blockID := &prototype.Sha256{Hash: id.Data[:]}

	dgpo := c.GetProps()
	dgpo.HeadBlockNumber = blk.Id().BlockNum()
	dgpo.HeadBlockId = blockID
	dgpo.Time = blk.SignedHeader.Header.Timestamp
	//c.dgpo.CurrentAslot       = c.dgpo.CurrentAslot + missedBlock+1

	// this check is useful ?
	mustSuccess(dgpo.GetHeadBlockNumber()-dgpo.GetIrreversibleBlockNum() < constants.MAX_UNDO_HISTORY, "The database does not have enough undo history to support a blockchain with so many missed blocks.")
	c.updateGlobalDataToDB(dgpo)
}

func (c *TrxPool) updateSigningWitness(blk *prototype.SignedBlock) {
	/*newAsLot := c.dgpo.GetCurrentAslot() + c.GetIncrementSlotAtTime(blk.SignedHeader.Header.Timestamp)

	name := blk.SignedHeader.Header.Witness
	witnessWrap := table.NewSoWitnessWrap(c.db, name)
	witnessWrap.MdLastConfirmedBlockNum(uint32(blk.Id().BlockNum()))
	witnessWrap.MdLastAslot(newAsLot)*/
}

func (c *TrxPool) createBlockSummary(blk *prototype.SignedBlock) {
	blockNum := blk.Id().BlockNum()
	blockNumSuffix := uint32(blockNum & 0x7ff)

	blockSummaryWrap := table.NewSoBlockSummaryObjectWrap(c.db, &blockNumSuffix)
	mustSuccess(blockSummaryWrap.CheckExist(), "can not get block summary object")
	blockIDArray := blk.Id().Data
	blockID := &prototype.Sha256{Hash: blockIDArray[:]}
	mustSuccess(blockSummaryWrap.MdBlockId(blockID), "update block summary object error")
}

func (c *TrxPool) clearExpiredTransactions() {
	sortWrap := table.STransactionObjectExpirationWrap{Dba: c.db}
	sortWrap.ForEachByOrder(nil, nil,
		func(mVal *prototype.Sha256, sVal *prototype.TimePointSec, idx uint32) bool {
			if sVal != nil {
				headTime := c.headBlockTime().UtcSeconds
				if headTime > sVal.UtcSeconds {
					// delete trx ...
					k := mVal
					objWrap := table.NewSoTransactionObjectWrap(c.db, k)
					mustSuccess(objWrap.RemoveTransactionObject(), "RemoveTransactionObject error")
				}
				return true
			}
			return false
		})
}

func (c *TrxPool) GetWitnessTopN(n uint32) []string {
	ret := []string{}
	revList := table.SWitnessVoteCountWrap{Dba: c.db}
	revList.ForEachByRevOrder(nil, nil, func(mVal *prototype.AccountName, sVal *uint64, idx uint32) bool {
		if mVal != nil {
			ret = append(ret, mVal.Value)
		}
		if idx < n {
			return true
		}
		return false
	})
	return ret
}

func (c *TrxPool) SetShuffledWitness(names []string) {
	witnessScheduleWrap := table.NewSoWitnessScheduleObjectWrap(c.db, &SingleId)
	mustSuccess(witnessScheduleWrap.MdCurrentShuffledWitness(names), "SetWitness error")
}

func (c *TrxPool) GetShuffledWitness() []string {
	witnessScheduleWrap := table.NewSoWitnessScheduleObjectWrap(c.db, &SingleId)
	return witnessScheduleWrap.GetCurrentShuffledWitness()
}

func (c *TrxPool) AddWeightedVP(value uint64) {
	dgpo := c.GetProps()
	dgpo.WeightedVps += value
	dgpWrap := table.NewSoGlobalWrap(c.db, &SingleId)
	dgpWrap.MdProps(dgpo)
}

func (c *TrxPool) saveReversion(num uint64) {
	tag := strconv.FormatUint(num, 10)
	currentRev := c.db.GetRevision()
	mustNoError(c.db.TagRevision(currentRev, tag), fmt.Sprintf("TagRevision:  tag:%d, reversion%d", num, currentRev))
	//c.log.GetLog().Debug("### saveReversion, num:", num, " rev:", currentRev)
}

func (c *TrxPool) getReversion(num uint64) uint64 {
	tag := strconv.FormatUint(num, 10)
	rev, err := c.db.GetTagRevision(tag)
	mustNoError(err, fmt.Sprintf("GetTagRevision: tag:%d, reversion:%d", num, rev))
	return rev
}

func (c *TrxPool) PopBlockTo(num uint64) {
	// undo pending trx
	c.ClearPending()
	/*if c.havePendingTransaction {
		mustNoError(c.db.EndTransaction(false), "EndTransaction error")
		c.havePendingTransaction = false
		//c.log.GetLog().Debug("@@@@@@ PopBlockTo havePendingTransaction=false")
	}*/
	// get reversion
	rev := c.getReversion(num)
	mustNoError(c.db.RevertToRevision(rev), fmt.Sprintf("RebaseToRevision error: tag:%d, reversion:%d", num, rev))
}

func (c *TrxPool) Commit(num uint64) {
	// this block can not be revert over, so it's irreversible
	rev := c.getReversion(num)
	//c.log.GetLog().Debug("### Commit, tag:", num, " rev:", rev)
	//c.log.GetLog().Debug("$$$ dump reversion array:",c.numToRev)
	mustNoError(c.db.RebaseToRevision(rev), fmt.Sprintf("RebaseToRevision: tag:%d, reversion:%d", num, rev))
}
