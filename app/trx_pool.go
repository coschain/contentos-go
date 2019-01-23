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
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"strconv"
	"github.com/coschain/contentos-go/common/crypto/secp256k1"
	"github.com/coschain/contentos-go/common/crypto"
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
	log     *logrus.Logger
	noticer EventBus.Bus
	skip    prototype.SkipFlag

	pendingTx []*prototype.EstimateTrxResult

	// TODO delete ??
	isProducing bool
	//currentTrxId           *prototype.Sha256
	//currentOpInTrx         uint16
	//currentBlockNum        uint64
	//currentTrxInBlock      int16
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

func (c *TrxPool) SetLog(log *logrus.Logger) {
	c.log = log
}

// service constructor
func NewController(ctx *node.ServiceContext, lg *logrus.Logger) (*TrxPool, error) {
	if lg == nil {
		lg = logrus.New()
		lg.SetOutput(ioutil.Discard)
	}
	return &TrxPool{ctx: ctx, log: lg}, nil
}

func (c *TrxPool) Start(node *node.Node) error {
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

		mustNoError(c.db.DeleteAll(), "truncate database error", prototype.StatusErrorDbTruncate)

		//c.log.Info("start initGenesis")
		c.initGenesis()
		c.saveReversion(0)
		//c.log.Info("finish initGenesis")
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

	trxWrp := &prototype.EstimateTrxResult{}
	trxWrp.SigTrx = trx
	trxWrp.Receipt = &prototype.TransactionReceiptWithInfo{}

	c.pendingTx = append(c.pendingTx, trxWrp)
}

func (c *TrxPool) PushTrx(trx *prototype.SignedTransaction) (invoice *prototype.TransactionReceiptWithInfo) {
	// this function may be cross routines ? use channel or lock ?
	oldSkip := c.skip
	defer func() {
		if err := recover(); err != nil {
			// !
			invoice = &prototype.TransactionReceiptWithInfo{}
			switch x := err.(type) {
			case prototype.Exception:
				invoice.Status = uint32(x.ErrorType)
				invoice.ErrorInfo = x.ToString()
			default:
				invoice.Status = prototype.StatusError
				invoice.ErrorInfo = "unknown error type"
			}
			c.log.Errorf("PushTrx Error: %v", err)
		}
		c.setProducing(false)
		c.skip = oldSkip
	}()

	// check maximum_block_size
	mustSuccess(proto.Size(trx) <= int(c.GetProps().MaximumBlockSize-256), "transaction is too large",prototype.StatusErrorTrxSize)

	c.setProducing(true)
	return c.pushTrx(trx)
}

func (c *TrxPool) GetProps() *prototype.DynamicProperties {
	dgpWrap := table.NewSoGlobalWrap(c.db, &SingleId)
	return dgpWrap.GetProps()
}

func (c *TrxPool) pushTrx(trx *prototype.SignedTransaction) *prototype.TransactionReceiptWithInfo {
	trxEst := &prototype.EstimateTrxResult{}
	trxEst.SigTrx = trx
	trxEst.Receipt = &prototype.TransactionReceiptWithInfo{}
	trxEst.Receipt.Status = prototype.StatusSuccess
	trxContext := NewTrxContext(trxEst, c.db, c)

	defer func() {
		// undo sub session
		if err := recover(); err != nil {
			mustNoError(c.db.EndTransaction(false), "EndTransaction error",prototype.StatusErrorDbEndTrx)
			switch x := err.(type) {
			case prototype.Exception:
				if x.ErrorType != prototype.StatusDeductGas {
					panic(err)
				} else {
					c.db.BeginTransaction()
					trxContext.DeductAllGasFee()
					mustNoError(c.db.EndTransaction(true), "EndTransaction error",prototype.StatusErrorDbEndTrx)
					c.pendingTx = append(c.pendingTx, trxEst)
					trxEst.Receipt.Status = prototype.StatusDeductGas
				}
			}
		} else {
			c.db.BeginTransaction()
			trxContext.DeductAllGasFee()
			mustNoError(c.db.EndTransaction(true), "EndTransaction error",prototype.StatusErrorDbEndTrx)
		}
	}()

	// start a new undo session when first transaction come after push block
	if !c.havePendingTransaction {
		tag := c.getBlockTag(uint64(c.headBlockNum())+1)
		c.db.BeginTransactionWithTag(tag)
		//	logging.CLog().Debug("@@@@@@ pushTrx havePendingTransaction=true")
		c.havePendingTransaction = true
	}

	// start a sub undo session for transaction
	c.db.BeginTransaction()
	c.applyTransactionInner(trxEst,trxContext)
	c.pendingTx = append(c.pendingTx, trxEst)

	// commit sub session
	mustNoError(c.db.EndTransaction(true), "EndTransaction error",prototype.StatusErrorDbEndTrx)

	// @ not use yet
	//c.notifyTrxPending(trx)
	return trxEst.Receipt
}

func (c *TrxPool) PushBlock(blk *prototype.SignedBlock, skip prototype.SkipFlag) error {
	var err error = nil
	oldFlag := c.skip
	c.skip = skip

	tmpPending := c.ClearPending()

	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {

			case prototype.Exception:
				err = errors.New(x.ToString())
			case error:
				err = x
				//c.log.Errorf("push block error : %v", x.Error())
			case string:
				err = errors.New(x)
				//c.log.Errorf("push block error : %v ", x)
			default:
				err = errors.New("unknown error type")
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
		tag := c.getBlockTag(blk.Id().BlockNum())
		c.db.BeginTransactionWithTag(tag)
		c.db.BeginTransaction()
		c.applyBlock(blk, skip)
		mustNoError(c.db.EndTransaction(true), "EndTransaction error",prototype.StatusErrorDbEndTrx)
	} else {
		// we have do a BeginTransaction at GenerateBlock
		c.applyBlock(blk, skip)
		mustNoError(c.db.EndTransaction(true), "EndTransaction error",prototype.StatusErrorDbEndTrx)
		c.havePendingTransaction = false
	}

	blockNum := blk.Id().BlockNum()
	c.saveReversion(blockNum)
	return err
}

func (c *TrxPool) ClearPending() []*prototype.EstimateTrxResult {
	// @
	mustSuccess(len(c.pendingTx) == 0 || c.havePendingTransaction, "can not clear pending",prototype.StatusErrorTrxClearPending)
	res := make([]*prototype.EstimateTrxResult, len(c.pendingTx))
	copy(res, c.pendingTx)

	c.pendingTx = c.pendingTx[:0]

	// 1. block from network, we undo pending state
	// 2. block from local generate, we keep it
	if c.skip&prototype.Skip_apply_transaction == 0 {
		if c.havePendingTransaction == true {
			mustNoError(c.db.EndTransaction(false), "EndTransaction error",prototype.StatusErrorDbEndTrx)
			c.havePendingTransaction = false
			//		c.log.Debug("@@@@@@ ClearPending havePendingTransaction=false")
		}
	}

	return res
}

func (c *TrxPool) restorePending(pending []*prototype.EstimateTrxResult) {
	for _, tw := range pending {
		id, err := tw.SigTrx.Id()
		mustNoError(err, "get transaction id error",prototype.StatusErrorTrxId)

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
			mustNoError(c.db.EndTransaction(false), "EndTransaction error",prototype.StatusErrorDbEndTrx)
			//c.log.Errorf("GenerateBlock Error: %v", err)
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
	mustNoError(err, "get public key error",prototype.StatusErrorTrxPriKeyToPubKey)

	witnessWrap := table.NewSoWitnessWrap(c.db, &prototype.AccountName{Value: witness})
	mustSuccess(bytes.Equal(witnessWrap.GetSigningKey().Data[:], pubkey.Data[:]), "public key not equal",prototype.StatusErrorTrxPubKeyCmp)

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
	//c.currentTrxInBlock = 0

	// undo all pending in DB
	if c.havePendingTransaction {
		mustNoError(c.db.EndTransaction(false), "EndTransaction error",prototype.StatusErrorDbEndTrx)
	}
	tag := c.getBlockTag(uint64(c.headBlockNum())+1)
	c.db.BeginTransactionWithTag(tag)
	c.db.BeginTransaction()
	//c.log.Debug("@@@@@@ GeneratBlock havePendingTransaction=true")
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
			trxContext := NewTrxContext(trxWraper, c.db, c)
			defer func() {
				if err := recover(); err != nil {
					mustNoError(c.db.EndTransaction(false), "EndTransaction error",prototype.StatusErrorDbEndTrx)
					switch x := err.(type) {
					case prototype.Exception:
						if x.ErrorType == prototype.StatusDeductGas {
							c.db.BeginTransaction()
							trxContext.DeductAllGasFee()
							mustNoError(c.db.EndTransaction(true), "EndTransaction error",prototype.StatusErrorDbEndTrx)
							totalSize += uint32(proto.Size(trxWraper))
							signBlock.Transactions = append(signBlock.Transactions, trxWraper.ToTrxWrapper())
						}
					}
				} else {
					c.db.BeginTransaction()
					trxContext.DeductAllGasFee()
					mustNoError(c.db.EndTransaction(true), "EndTransaction error",prototype.StatusErrorDbEndTrx)
					totalSize += uint32(proto.Size(trxWraper))
					signBlock.Transactions = append(signBlock.Transactions, trxWraper.ToTrxWrapper())
				}
			}()

			c.db.BeginTransaction()
			c.applyTransactionInner(trxWraper,trxContext)
			mustNoError(c.db.EndTransaction(true), "EndTransaction error",prototype.StatusErrorDbEndTrx)

			totalSize += uint32(proto.Size(trxWraper))
			signBlock.Transactions = append(signBlock.Transactions, trxWraper.ToTrxWrapper())
			//c.currentTrxInBlock++
		}()
	}
	if postponeTrx > 0 {
		//c.log.Warnf("postponed %d trx due to max block size", postponeTrx)
	}

	signBlock.SignedHeader.Header.Previous = pre
	signBlock.SignedHeader.Header.Timestamp = &prototype.TimePointSec{UtcSeconds: timestamp}
	id := signBlock.CalculateMerkleRoot()
	signBlock.SignedHeader.Header.TransactionMerkleRoot = &prototype.Sha256{Hash: id.Data[:]}
	signBlock.SignedHeader.Header.Witness = &prototype.AccountName{Value: witness}
	signBlock.SignedHeader.WitnessSignature = &prototype.SignatureType{}
	signBlock.SignedHeader.Sign(priKey)

	mustSuccess(proto.Size(signBlock) <= constants.MAX_BLOCK_SIZE, "block size too big",prototype.StatusErrorTrxMaxBlockSize)
	// clearpending then let dpos call PushBlock, the point is without restore pending step when PushBlock
	//c.ClearPending()

	/*mustNoError(c.db.EndTransaction(false), "EndTransaction error")
	c.havePendingTransaction = false*/
	//c.log.Debug("@@@@@@ GenerateBlock havePendingTransaction=false")

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

func (c *TrxPool) applyTransaction(trxEst *prototype.EstimateTrxResult,trxContext *TrxContext) {
	c.applyTransactionInner(trxEst,trxContext)
	// @ not use yet
	//c.notifyTrxPostExecute(trxWrp.SigTrx)
}

func (c *TrxPool) applyTransactionInner(trxEst *prototype.EstimateTrxResult,trxContext *TrxContext) {
	defer func() {
		useGas := trxContext.HasGasFee()
		if err := recover();err != nil {
			if useGas {
				e := &prototype.Exception{HelpString:"apply transaction failed",ErrorType:prototype.StatusDeductGas}
				panic(e)
			} else {
				panic(err)
			}
		} else {
			return
		}
	}()

	trx := trxEst.SigTrx
	var err error
	currentTrxId, err := trx.Id()
	mustNoError(err, "get trx id failed", prototype.StatusErrorTrxId)

	trx.Validate()

	// trx duplicate check
	transactionObjWrap := table.NewSoTransactionObjectWrap(c.db, currentTrxId)
	mustSuccess(!transactionObjWrap.CheckExist(), "Duplicate transaction check failed",prototype.StatusErrorTrxDuplicateCheck)

	if c.skip&prototype.Skip_transaction_signatures == 0 {
		tmpChainId := prototype.ChainId{Value: 0}
		mustNoError(trxContext.InitSigState(tmpChainId), "signature export error", prototype.StatusErrorTrxExportPubKey)
		trxContext.VerifySignature()
		// @ check_admin
	}

	blockNum := c.GetProps().GetHeadBlockNumber()
	if blockNum > 0 {
		uniWrap := table.UniBlockSummaryObjectIdWrap{Dba: c.db}
		idWrap := uniWrap.UniQueryId(&trx.Trx.RefBlockNum)
		if !idWrap.CheckExist() {
			e := &prototype.Exception{HelpString:"TaPos RefBlockNum missing",ErrorType:prototype.StatusErrorTrxTaPos}
			panic(e)
		} else {
			blockId := idWrap.GetBlockId()
			summaryId := binary.BigEndian.Uint32(blockId.Hash[8:12])
			mustSuccess(trx.Trx.RefBlockPrefix == summaryId, "TaPos RefBlockNum not equal",prototype.StatusErrorTrxTaPos)
		}

		now := c.GetProps().Time
		// get head time
		mustSuccess(trx.Trx.Expiration.UtcSeconds <= uint32(now.UtcSeconds+constants.TRX_MAX_EXPIRATION_TIME), "transaction expiration too long",prototype.StatusErrorTrxExpire)
		mustSuccess(now.UtcSeconds < trx.Trx.Expiration.UtcSeconds, "transaction has expired",prototype.StatusErrorTrxExpire)
	}

	// insert trx into DB unique table
	cErr := transactionObjWrap.Create(func(tInfo *table.SoTransactionObject) {
		tInfo.TrxId = currentTrxId
		tInfo.Expiration = trx.Trx.Expiration
	})
	mustNoError(cErr, "create transactionObject failed", prototype.StatusErrorDbCreate)

	// @ not use yet
	//c.notifyTrxPreExecute(trx)

	// process operation
	//c.currentOpInTrx = 0
	for _, op := range trx.Trx.Operations {
		c.applyOperation(trxContext, op)
		//c.currentOpInTrx++
	}

	//c.currentTrxId = &prototype.Sha256{}
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
	//nextBlockNum := blk.Id().BlockNum()

	merkleRoot := blk.CalculateMerkleRoot()
	mustSuccess(bytes.Equal(merkleRoot.Data[:], blk.SignedHeader.Header.TransactionMerkleRoot.Hash), "Merkle check failed", prototype.StatusErrorTrxMerkleCheck)

	// validate_block_header
	c.validateBlockHeader(blk)

	//c.currentBlockNum = nextBlockNum
	//c.currentTrxInBlock = 0

	blockSize := proto.Size(blk)
	mustSuccess(uint32(blockSize) <= c.GetProps().GetMaximumBlockSize(), "Block size is too big",prototype.StatusErrorTrxMaxBlockSize)

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

	trxEst := &prototype.EstimateTrxResult{}
	trxEst.Receipt = &prototype.TransactionReceiptWithInfo{}

	if skip&prototype.Skip_apply_transaction == 0 {

		for _, tw := range blk.Transactions {
			func () {
				trxContext := NewTrxContext(trxEst, c.db, c)
				defer func() {
					if err := recover();err != nil {
						switch x := err.(type) {
						case prototype.Exception:
							if x.ErrorType != prototype.StatusDeductGas {
								panic(err)
							} else {
								c.db.BeginTransaction()
								trxContext.DeductAllGasFee()
								mustNoError(c.db.EndTransaction(true), "EndTransaction error",prototype.StatusErrorDbEndTrx)
							}
						}
					} else {
						c.db.BeginTransaction()
						trxContext.DeductAllGasFee()
						mustNoError(c.db.EndTransaction(true), "EndTransaction error",prototype.StatusErrorDbEndTrx)
					}
				}()
				trxEst.SigTrx = tw.SigTrx
				trxEst.Receipt.Status = prototype.StatusSuccess
				c.applyTransaction(trxEst, trxContext)
				mustSuccess(trxEst.Receipt.Status == tw.Invoice.Status, "mismatched invoice", prototype.StatusErrorTrxApplyInvoice)
				//c.currentTrxInBlock++
			}()
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
			mustNoError(c.db.EndTransaction(false), "EndTransaction error", prototype.StatusErrorDbEndTrx)
			panic(err)
		} else {
			mustNoError(c.db.EndTransaction(true), "EndTransaction error", prototype.StatusErrorDbEndTrx)
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
	}), "CreateAccount error", prototype.StatusErrorDbCreate)

	// create account authority
	authorityWrap := table.NewSoAccountAuthorityObjectWrap(c.db, name)
	ownerAuth := prototype.NewAuthorityFromPubKey(pubKey)

	mustNoError(authorityWrap.Create(func(tInfo *table.SoAccountAuthorityObject) {
		tInfo.Account = name
		tInfo.Owner = ownerAuth
	}), "CreateAccountAuthorityObject error ", prototype.StatusErrorDbCreate)

	// create witness_object
	witnessWrap := table.NewSoWitnessWrap(c.db, name)
	mustNoError(witnessWrap.Create(func(tInfo *table.SoWitness) {
		tInfo.Owner = name
		tInfo.WitnessScheduleType = &prototype.WitnessScheduleType{Value: prototype.WitnessScheduleType_miner}
		tInfo.CreatedTime = &prototype.TimePointSec{UtcSeconds: 0}
		tInfo.SigningKey = pubKey
		tInfo.LastWork = &prototype.Sha256{Hash: []byte{0}}
	}), "Witness Create Error", prototype.StatusErrorDbCreate)

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
	}), "CreateDynamicGlobalProperties error", prototype.StatusErrorDbCreate)

	//create rewards keeper
	keeperWrap := table.NewSoRewardsKeeperWrap(c.db, &SingleId)
	rewards := make(map[string]*prototype.Vest)
	rewards["initminer"] = &prototype.Vest{Value: 0}
	mustNoError(keeperWrap.Create(func(tInfo *table.SoRewardsKeeper) {
		tInfo.Id = SingleId
		//tInfo.Keeper.Rewards = map[string]*prototype.Vest{}
		tInfo.Keeper = &prototype.InternalRewardsKeeper{Rewards: rewards}
	}), "Create Rewards Keeper error", prototype.StatusErrorDbCreate)

	// create block summary buffer 2048
	for i := uint32(0); i < 0x800; i++ {
		wrap := table.NewSoBlockSummaryObjectWrap(c.db, &i)
		mustNoError(wrap.Create(func(tInfo *table.SoBlockSummaryObject) {
			tInfo.Id = i
			tInfo.BlockId = &prototype.Sha256{Hash: make([]byte, 32)}
		}), "CreateBlockSummaryObject error", prototype.StatusErrorDbCreate)
	}

	// create witness scheduler
	witnessScheduleWrap := table.NewSoWitnessScheduleObjectWrap(c.db, &SingleId)
	mustNoError(witnessScheduleWrap.Create(func(tInfo *table.SoWitnessScheduleObject) {
		tInfo.Id = SingleId
		tInfo.CurrentShuffledWitness = append(tInfo.CurrentShuffledWitness, constants.COS_INIT_MINER)
	}), "CreateWitnessScheduleObject error", prototype.StatusErrorDbCreate)
}

func (c *TrxPool) TransferToVest(value *prototype.Coin) {

	dgpo := c.GetProps()
	cos := dgpo.GetTotalCos()
	vest := dgpo.GetTotalVestingShares()
	addVest := value.ToVest()

	mustNoError(cos.Sub(value), "TotalCos overflow", prototype.StatusErrorTrxOverflow)
	dgpo.TotalCos = cos

	mustNoError(vest.Add(addVest), "TotalVestingShares overflow", prototype.StatusErrorTrxOverflow)
	dgpo.TotalVestingShares = vest

	c.updateGlobalDataToDB(dgpo)
}

func (c *TrxPool) TransferFromVest(value *prototype.Vest) {
	dgpo := c.GetProps()

	cos := dgpo.GetTotalCos()
	vest := dgpo.GetTotalVestingShares()
	addCos := value.ToCoin()

	mustNoError(cos.Add(addCos), "TotalCos overflow", prototype.StatusErrorTrxOverflow)
	dgpo.TotalCos = cos

	mustNoError(vest.Sub(value), "TotalVestingShares overflow", prototype.StatusErrorTrxOverflow)
	dgpo.TotalVestingShares = vest

	// TODO if op execute failed ???? how to revert ??
	c.updateGlobalDataToDB(dgpo)
}

func (c *TrxPool) validateBlockHeader(blk *prototype.SignedBlock) {
	headID := c.headBlockID()
	if !bytes.Equal(headID.Hash, blk.SignedHeader.Header.Previous.Hash) {
		e := &prototype.Exception{HelpString:"hash not equal",ErrorType:prototype.StatusErrorTrxBlockHeaderCheck}
		panic(e)
	}
	headTime := c.headBlockTime()
	if headTime.UtcSeconds >= blk.SignedHeader.Header.Timestamp.UtcSeconds {
		e := &prototype.Exception{HelpString:"block time is invalid",ErrorType:prototype.StatusErrorTrxBlockHeaderCheck}
		panic(e)
	}

	// witness sig check
	witnessName := blk.SignedHeader.Header.Witness
	witnessWrap := table.NewSoWitnessWrap(c.db, witnessName)
	pubKey := witnessWrap.GetSigningKey()
	res, err := blk.SignedHeader.ValidateSig(pubKey)
	if !res || err != nil {
		e := &prototype.Exception{ErrorString:err.Error(),HelpString:"block signature check error",ErrorType:prototype.StatusErrorTrxBlockHeaderCheck}
		panic(e)
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
	mustSuccess(dgpWrap.MdProps(dgpo), "update global data error",prototype.StatusErrorDbUpdate)
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
	mustSuccess(dgpo.GetHeadBlockNumber()-dgpo.GetIrreversibleBlockNum() < constants.MAX_UNDO_HISTORY, "The database does not have enough undo history to support a blockchain with so many missed blocks.",prototype.StatusErrorTrxMaxUndo)
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
	mustSuccess(blockSummaryWrap.CheckExist(), "can not get block summary object",prototype.StatusErrorDbExist)
	blockIDArray := blk.Id().Data
	blockID := &prototype.Sha256{Hash: blockIDArray[:]}
	mustSuccess(blockSummaryWrap.MdBlockId(blockID), "update block summary object error",prototype.StatusErrorDbUpdate)
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
					mustSuccess(objWrap.RemoveTransactionObject(), "RemoveTransactionObject error",prototype.StatusErrorDbDelete)
				}
				return true
			}
			return false
		})
}

func (c *TrxPool) GetWitnessTopN(n uint32) []string {
	var ret []string
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
	mustSuccess(witnessScheduleWrap.MdCurrentShuffledWitness(names), "SetWitness error",prototype.StatusErrorDbUpdate)
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
	mustNoError(c.db.TagRevision(currentRev, tag), fmt.Sprintf("TagRevision:  tag:%d, reversion%d", num, currentRev), prototype.StatusErrorDbTag)
	//c.log.Debug("### saveReversion, num:", num, " rev:", currentRev)
}

func (c *TrxPool) getReversion(num uint64) uint64 {
	tag := strconv.FormatUint(num, 10)
	rev, err := c.db.GetTagRevision(tag)
	mustNoError(err, fmt.Sprintf("GetTagRevision: tag:%d, reversion:%d", num, rev), prototype.StatusErrorDbTag)
	return rev
}

func (c *TrxPool) getBlockTag(num uint64) string {
	tag := strconv.FormatUint(num, 10)
	return tag
}

func (c *TrxPool) PopBlockTo(num uint64) {
	// undo pending trx
	c.ClearPending()
	/*if c.havePendingTransaction {
		mustNoError(c.db.EndTransaction(false), "EndTransaction error")
		c.havePendingTransaction = false
		//c.log.Debug("@@@@@@ PopBlockTo havePendingTransaction=false")
	}*/
	// get reversion
	//rev := c.getReversion(num)
	//mustNoError(c.db.RevertToRevision(rev), fmt.Sprintf("RebaseToRevision error: tag:%d, reversion:%d", num, rev))
	tag := c.getBlockTag(num)
	mustSuccess(c.db.RollBackToTag(tag)==nil, fmt.Sprintf("RevertToRevision error: tag:%d", num,),prototype.StatusErrorDbTag)
}

func (c *TrxPool) Commit(num uint64) {
	// this block can not be revert over, so it's irreversible
	tag := c.getBlockTag(uint64(num))
	err := c.db.Squash(tag,num)
	mustSuccess(err == nil,fmt.Sprintf("SquashBlock: tag:%d,error is %s",num,err),prototype.StatusErrorDbTag)
}

func (c *TrxPool) VerifySig(name *prototype.AccountName, digest []byte, sig []byte) bool {
	// public key from db
	witnessWrap := table.NewSoWitnessWrap(c.db, name)
	if !witnessWrap.CheckExist() {
		return false
	}
	dbPubKey := witnessWrap.GetSigningKey()
	if dbPubKey == nil {
		return false
	}

	// public key from parameter
	buffer, err := secp256k1.RecoverPubkey(digest, sig)

	if err != nil {
		return false
	}

	ecPubKey, err := crypto.UnmarshalPubkey(buffer)
	if err != nil {
		return false
	}
	keyBuffer := secp256k1.CompressPubkey(ecPubKey.X, ecPubKey.Y)
	result := new(prototype.PublicKeyType)
	result.Data = keyBuffer

	// compare bytes
	if bytes.Equal(dbPubKey.Data,result.Data) {
		return true
	}

	return false
}

func (c *TrxPool) Sign(priv *prototype.PrivateKeyType, digest []byte) []byte {
	res, err := secp256k1.Sign(digest[:], priv.Data)
	if err != nil {
		return nil
	}
	return res
}

//Sync blocks to squash db when node reStart
//pushedBlk: the already pushed blocks
//commitBlk: the already commit blocks what are stored in block log
//realCommit: the latest commit number of block in block log
//headBlk: the head block in main chain
func (c *TrxPool) SyncBlockDataToDB (pushedBlk []common.ISignedBlock, commitBlk []common.ISignedBlock,
	realCommit uint64, headBlk common.ISignedBlock) {
	if headBlk == nil {
		return
	}

	//Fetch the latest commit block number in squash db
	num,err := c.db.GetCommitNum()
	if err != nil {
		c.log.Debug("[sync Block]: Failed to get latest commit block number")
		return
	}

	//Fetch the real commit block number in chain
	realNum := realCommit
	//Fetch the head block number in chain
	headNum := headBlk.Id().BlockNum()

	//syncNum is the start number need to fetch from ForkDB
	syncNum := num

	//Reload lost commit blocks
	if realNum > 0 && num < realNum {
		cnt := uint64(len(commitBlk))
		if cnt > 0  {
			//Scene 1: The block data in squash db has been deleted,so need sync lost blocks to squash db
			//Because ForkDB will not continue to store commit blocks,so we need load all commit blocks from block log,
			//meanWhile add block to squash db
			var start = num
			if start >= cnt {
				c.log.Errorf("[Reload commit] start index %v out range of reload block count %v",start,cnt)
			}else {
				c.log.Debugf("[Reload commit] start sync lost commit blocks from block log,db commit num is: " +
					"%v,end:%v,real commit num is %v", start, headNum, realNum)
				for i := start ; i < uint64(cnt); i++ {
					blk := commitBlk[i]
					c.log.Debugf("[Reload commit] push block,blockNum is: " +
						"%v", blk.(*prototype.SignedBlock).Id().BlockNum())
					err = c.PushBlock(blk.(*prototype.SignedBlock),prototype.Skip_nothing)
					if err != nil {
						desc := fmt.Sprintf("[Reload commit] push the block which num is %v fail,error " +
							"is %s", i, err)
						panic(desc)
					}
				}
			}
			syncNum = uint64(cnt)
		}else {
			c.log.Errorf("[Reload commit] Failed to get lost commit blocks from block log," +
				"start: %v," + "end:%v,real commit num is %v", syncNum+1, headNum, realNum)
		}

	}

	//1.Synchronous all the lost pushed blocks from ForkDB to squash db
	if headNum > syncNum {
		if headNum-syncNum > uint64(len(pushedBlk)) {
			desc := fmt.Sprintf("[sync pushed]: the lost blocks range %v from forkDB is less than " +
				"real lost range [%v %v] ",len(pushedBlk), syncNum+1, headNum)
			panic(desc)
		}
		c.log.Debugf("[sync pushed]: start sync lost blocks,start: %v,end:%v,real commit num " +
			"is %v", syncNum+1, headNum, realNum)
		for i := range pushedBlk {
			blk := pushedBlk[i]
			c.log.Debugf("[sync pushed]: sync pushed block,blockNum is: " +
				"%v", blk.(*prototype.SignedBlock).Id().BlockNum())
			err = c.PushBlock(blk.(*prototype.SignedBlock),prototype.Skip_nothing)
			if err != nil {
				desc := fmt.Sprintf("[sync pushed]: push the block which num is %v fail,error is %s", i, err)
				panic(desc)
			}
		}
	}

	//2.Synchronous the lost commit blocks to squash db
	if realNum > num {
		//Need synchronous commit
		c.log.Debugf("[sync commit] start sync commit block num %v ",realNum)
		c.Commit(realNum)
	}
}
