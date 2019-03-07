package app

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/common/crypto"
	"github.com/coschain/contentos-go/common/crypto/secp256k1"
	"github.com/coschain/contentos-go/common/eventloop"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/utils"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"time"
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

	pendingTx []*prototype.TransactionWrapper

	// TODO delete ??
	isProducing bool
	//currentTrxId           *prototype.Sha256
	//currentOpInTrx         uint16
	//currentBlockNum        uint64
	//currentTrxInBlock      int16
	havePendingTransaction bool
	shuffle                common.ShuffleFunc

	iceberg         *BlockIceberg
	resourceLimiter utils.IResourceLimiter
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
	c.iceberg = NewBlockIceberg(c.db)
	dgpWrap := table.NewSoGlobalWrap(c.db, &constants.GlobalId)
	if !dgpWrap.CheckExist() {

		mustNoError(c.db.DeleteAll(), "truncate database error", prototype.StatusErrorDbTruncate)

		//c.log.Info("start initGenesis")
		c.initGenesis()

		mustNoError(c.db.TagRevision(c.db.GetRevision(), GENESIS_TAG), "genesis tagging failed", prototype.StatusErrorDbTag)
		c.iceberg = NewBlockIceberg(c.db)

		//c.log.Info("finish initGenesis")
	}
	c.resourceLimiter = utils.NewResourceLimiter(c.db)
}

func (c *TrxPool) Stop() error {
	return nil
}

func (c *TrxPool) setProducing(b bool) {
	c.isProducing = b
}

func (c *TrxPool) PushTrxToPending(trx *prototype.SignedTransaction) (err error) {
	defer func() {
		if r := recover(); r != nil {
			switch val := r.(type) {
			case error:
				err = val
			case string:
				err = errors.New(val)
			default:
				err = errors.New("unknown panic type when push trx to pending list")
			}
		}
	}()
	c.addTrxToPending(trx, false)
	return err
}

func (c *TrxPool) addTrxToPending(trx *prototype.SignedTransaction, isVerified bool) {
	if !c.havePendingTransaction {
		c.db.BeginTransaction()
		c.havePendingTransaction = true
	}

	trxWrp := &prototype.TransactionWrapper{}
	trxWrp.SigTrx = trx
	trxWrp.Receipt = &prototype.TransactionReceipt{}

	if !isVerified {
		//verify the signature
		trxContext := NewTrxContext(trxWrp, c.db, c)
		trx.Validate()
		tmpChainId := prototype.ChainId{Value: 0}
		mustNoError(trxContext.InitSigState(tmpChainId), "signature export error", prototype.StatusError)
		trxContext.VerifySignature()
	}
	c.pendingTx = append(c.pendingTx, trxWrp)
}

func (c *TrxPool) PushTrx(trx *prototype.SignedTransaction) (receipt *prototype.TransactionReceipt) {
	// this function may be cross routines ? use channel or lock ?
	oldSkip := c.skip
	tw := &prototype.TransactionWrapper{}
	defer func() {
		if err := recover(); err != nil {
			switch x := err.(type) {
			case *prototype.Exception:
				tw.Receipt.Status = uint32(x.ErrorType)
				tw.Receipt.ErrorInfo = x.ToString()
			default:
				tw.Receipt.Status = prototype.StatusError
				tw.Receipt.ErrorInfo = "unknown error type"
			}
			c.log.Errorf("PushTrx Error: %v", err)
		}
		c.setProducing(false)
		c.skip = oldSkip
		receipt = tw.Receipt
	}()

	// check maximum_block_size
	mustSuccess(proto.Size(trx) <= int(c.GetProps().MaximumBlockSize-256), "transaction is too large", prototype.StatusErrorTrxSize)

	c.setProducing(true)

	tw.SigTrx = trx
	tw.Receipt = &prototype.TransactionReceipt{}
	tw.Receipt.Status = prototype.StatusSuccess

	c.pushTrx(tw)
	return tw.Receipt
}

func (c *TrxPool) GetProps() *prototype.DynamicProperties {
	dgpWrap := table.NewSoGlobalWrap(c.db, &constants.GlobalId)
	return dgpWrap.GetProps()
}

func (c *TrxPool) pushTrx(tw *prototype.TransactionWrapper) {
	trxContext := NewTrxContext(tw, c.db, c)
	defer func() {
		// undo sub session
		if err := recover(); err != nil {
			mustNoError(c.db.EndTransaction(false), "EndTransaction error", prototype.StatusErrorDbEndTrx)
			switch x := err.(type) {
			case *prototype.Exception:
				if x.ErrorType != prototype.StatusDeductGas {
					panic(err)
				}
			}
		}
		c.PayGas(trxContext)
		trxContext.Finalize()
		c.pendingTx = append(c.pendingTx, tw)
	}()

	// start a new undo session when first transaction come after push block
	if !c.havePendingTransaction {
		c.db.BeginTransaction()
		//	logging.CLog().Debug("@@@@@@ pushTrx havePendingTransaction=true")
		c.havePendingTransaction = true
	}

	// start a sub undo session for transaction
	c.db.BeginTransaction()
	c.applyTransactionInner(true, trxContext)

	// commit sub session
	mustNoError(c.db.EndTransaction(true), "EndTransaction error", prototype.StatusErrorDbEndTrx)

	// @ not use yet
	//c.notifyTrxPending(trx)
}

func (c *TrxPool) PushBlock(blk *prototype.SignedBlock, skip prototype.SkipFlag) (err error) {
	if blk == nil {
		return errors.New("block is nil")
	}
	//var err error = nil
	oldFlag := c.skip
	c.skip = skip

	tmpPending := c.ClearPending()

	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {

			case *prototype.Exception:
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
			c.iceberg.EndBlock(false)
			if skip&prototype.Skip_apply_transaction != 0 {
				c.havePendingTransaction = false
			}
			fmt.Printf("push block fail,the error is %v,the block num is %v \n", r, blk.Id().BlockNum())
		}
		// restorePending will call pushTrx, will start new transaction for pending
		c.restorePending(tmpPending)

		c.skip = oldFlag

	}()

	if skip&prototype.Skip_apply_transaction == 0 {
		c.iceberg.BeginBlock(blk.Id().BlockNum())
		c.db.BeginTransaction()
		c.applyBlock(blk, skip)
		mustNoError(c.db.EndTransaction(true), "EndTransaction error", prototype.StatusErrorDbEndTrx)
		c.iceberg.EndBlock(true)
	} else {
		// we have do a BeginTransaction at GenerateBlock
		c.applyBlock(blk, skip)
		mustNoError(c.db.EndTransaction(true), "EndTransaction error", prototype.StatusErrorDbEndTrx)
		c.iceberg.EndBlock(true)
		c.havePendingTransaction = false
	}

	return err
}

func (c *TrxPool) ClearPending() []*prototype.TransactionWrapper {
	// @
	mustSuccess(len(c.pendingTx) == 0 || c.havePendingTransaction, "can not clear pending", prototype.StatusErrorTrxClearPending)
	res := make([]*prototype.TransactionWrapper, len(c.pendingTx))
	copy(res, c.pendingTx)

	c.pendingTx = c.pendingTx[:0]

	// 1. block from network, we undo pending state
	// 2. block from local generate, we keep it
	if c.skip&prototype.Skip_apply_transaction == 0 {
		if c.havePendingTransaction == true {
			mustNoError(c.db.EndTransaction(false), "EndTransaction error", prototype.StatusErrorDbEndTrx)
			c.havePendingTransaction = false
			//		c.log.Debug("@@@@@@ ClearPending havePendingTransaction=false")
		}
	}

	return res
}

func (c *TrxPool) restorePending(pending []*prototype.TransactionWrapper) {
	for _, tw := range pending {
		id, err := tw.SigTrx.Id()
		mustNoError(err, "get transaction id error", prototype.StatusErrorTrxId)

		objWrap := table.NewSoTransactionObjectWrap(c.db, id)
		if !objWrap.CheckExist() {
			c.addTrxToPending(tw.SigTrx, true)
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
			mustNoError(c.db.EndTransaction(false), "EndTransaction error", prototype.StatusErrorDbEndTrx)
			//c.log.Errorf("GenerateBlock Error: %v", err)
		}
	}()

	c.skip = skip

	/*
		slotNum := c.GetIncrementSlotAtTime(&prototype.TimePointSec{UtcSeconds:timestamp})
		mustSuccess(slotNum > 0,"slot num must > 0")
		witnessName := c.GetScheduledWitness(slotNum)
		mustSuccess(witnessName.Value == witness,"not this witness")*/

	pubkey, err := priKey.PubKey()
	mustNoError(err, "get public key error", prototype.StatusErrorTrxPriKeyToPubKey)

	witnessWrap := table.NewSoWitnessWrap(c.db, &prototype.AccountName{Value: witness})
	mustSuccess(bytes.Equal(witnessWrap.GetSigningKey().Data[:], pubkey.Data[:]), "public key not equal", prototype.StatusErrorTrxPubKeyCmp)

	// @ signHeader size is zero, must have some content
	signHeader := &prototype.SignedBlockHeader{}
	emptyHeader(signHeader)
	maxBlockHeaderSize := proto.Size(signHeader) + 4

	dgpWrap := table.NewSoGlobalWrap(c.db, &constants.GlobalId)
	maxBlockSize := dgpWrap.GetProps().MaximumBlockSize
	var totalSize uint32 = uint32(maxBlockHeaderSize) + 2048 // block size will expand after sign

	signBlock := &prototype.SignedBlock{}
	signBlock.SignedHeader = &prototype.SignedBlockHeader{}
	signBlock.SignedHeader.Header = &prototype.BlockHeader{}
	//c.currentTrxInBlock = 0

	// undo all pending in DB
	if c.havePendingTransaction {
		mustNoError(c.db.EndTransaction(false), "EndTransaction error", prototype.StatusErrorDbEndTrx)
	}
	c.iceberg.BeginBlock(c.headBlockNum() + 1)
	c.db.BeginTransaction()
	//c.log.Debug("@@@@@@ GeneratBlock havePendingTransaction=true")
	c.havePendingTransaction = true

	var postponeTrx uint64 = 0
	isFinish := false
	time.AfterFunc(650*time.Millisecond, func() {
		isFinish = true
	})
	failTrxMap := make(map[int]int)

	for k, trxWraper := range c.pendingTx {
		if isFinish {
			break
		}
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
					failTrxMap[k] = k
					mustNoError(c.db.EndTransaction(false), "EndTransaction error", prototype.StatusErrorDbEndTrx)
					switch x := err.(type) {
					case *prototype.Exception:
						if x.ErrorType != prototype.StatusDeductGas {
							return
						}
					}
				}
				c.PayGas(trxContext)
				trxContext.Finalize()
				totalSize += uint32(proto.Size(trxWraper))
				signBlock.Transactions = append(signBlock.Transactions, trxWraper)
			}()

			c.db.BeginTransaction()
			c.applyTransactionInner(false, trxContext)
			mustNoError(c.db.EndTransaction(true), "EndTransaction error", prototype.StatusErrorDbEndTrx)
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

	size := proto.Size(signBlock)
	mustSuccess(size <= constants.MaxBlockSize, "block size too big", prototype.StatusErrorTrxMaxBlockSize)
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
	if len(failTrxMap) > 0 {
		copyPending := make([]*prototype.TransactionWrapper, 0, len(c.pendingTx))
		for k, v := range c.pendingTx {
			if _, ok := failTrxMap[k]; !ok {
				copyPending = append(copyPending, v)
			}
		}
		c.pendingTx = c.pendingTx[:0]
		c.pendingTx = append(c.pendingTx, copyPending...)

	}
	return signBlock
}

func (c *TrxPool) notifyOpPreExecute(on *prototype.OperationNotification) {
	c.noticer.Publish(constants.NoticeOpPre, on)
}

func (c *TrxPool) notifyOpPostExecute(on *prototype.OperationNotification) {
	c.noticer.Publish(constants.NoticeOpPost, on)
}

func (c *TrxPool) notifyTrxPreExecute(trx *prototype.SignedTransaction) {
	c.noticer.Publish(constants.NoticeTrxPre, trx)
}

func (c *TrxPool) notifyTrxPostExecute(trx *prototype.SignedTransaction) {
	c.noticer.Publish(constants.NoticeTrxPost, trx)
}

func (c *TrxPool) notifyTrxPending(trx *prototype.SignedTransaction) {
	c.noticer.Publish(constants.NoticeTrxPending, trx)
}

func (c *TrxPool) notifyBlockApply(block *prototype.SignedBlock) {
	c.noticer.Publish(constants.NoticeBlockApplied, block)
}

func (c *TrxPool) notifyTrxApplyResult(trx *prototype.SignedTransaction, res bool,
	receipt *prototype.TransactionReceipt) {
	c.noticer.Publish(constants.NoticeTrxApplied, trx, receipt)
}

func (c *TrxPool) applyTransaction(trxContext *TrxContext) {
	c.applyTransactionInner(c.skip&prototype.Skip_transaction_signatures == 0, trxContext)
	// @ not use yet
	//c.notifyTrxPostExecute(trxWrp.SigTrx)
}

func (c *TrxPool) applyTransactionInner(isNeedVerify bool, trxContext *TrxContext) {
	c.db.Lock()
	tw := trxContext.Wrapper
	defer func() {
		c.db.Unlock()

		useGas := trxContext.HasGasFee()
		if err := recover(); err != nil {

			if useGas {
				trxContext.SetStatus(prototype.StatusDeductGas)
				e := &prototype.Exception{HelpString: "apply transaction failed", ErrorType: prototype.StatusDeductGas}
				c.notifyTrxApplyResult(tw.SigTrx, true, tw.Receipt)
				panic(e)
			} else {
				e := err.(*prototype.Exception)
				trxContext.SetStatus(e.ErrorType)
				c.notifyTrxApplyResult(tw.SigTrx, false, tw.Receipt)
				panic(err)
			}
		} else {
			trxContext.SetStatus(prototype.StatusSuccess)
			trxContext.Finalize()
			c.notifyTrxApplyResult(tw.SigTrx, true, tw.Receipt)
			return
		}
	}()

	// check net resource
	if c.ctx.Config().ResourceCheck {
		trxContext.CheckNet(uint64(proto.Size(tw.SigTrx)))
	}

	trx := tw.SigTrx
	var err error
	currentTrxId, err := trx.Id()
	mustNoError(err, "get trx id failed", prototype.StatusErrorTrxId)

	trx.Validate()

	// trx duplicate check
	transactionObjWrap := table.NewSoTransactionObjectWrap(c.db, currentTrxId)
	mustSuccess(!transactionObjWrap.CheckExist(), "Duplicate transaction check failed", prototype.StatusErrorDbExist)

	if isNeedVerify {
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
			e := &prototype.Exception{HelpString: "TaPos RefBlockNum missing", ErrorType: prototype.StatusErrorTrxTaPos}
			panic(e)
		} else {
			blockId := idWrap.GetBlockId()
			summaryId := binary.BigEndian.Uint32(blockId.Hash[8:12])
			mustSuccess(trx.Trx.RefBlockPrefix == summaryId, "TaPos RefBlockNum not equal", prototype.StatusErrorTrxTaPos)
		}

		now := c.GetProps().Time
		// get head time
		mustSuccess(trx.Trx.Expiration.UtcSeconds <= uint32(now.UtcSeconds+constants.TrxMaxExpirationTime), "transaction expiration too long", prototype.StatusErrorTrxExpire)
		mustSuccess(now.UtcSeconds < trx.Trx.Expiration.UtcSeconds, "transaction has expired", prototype.StatusErrorTrxExpire)
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
	mustSuccess(uint32(blockSize) <= c.GetProps().GetMaximumBlockSize(), "Block size is too big", prototype.StatusErrorTrxMaxBlockSize)

	if uint32(blockSize) < constants.MinBlockSize {
		// elog("Block size is too small")
	}

	w := blk.SignedHeader.Header.Witness
	dgpo := c.GetProps()
	dgpo.CurrentWitness = w
	dgpWrap := table.NewSoGlobalWrap(c.db, &constants.GlobalId)
	dgpWrap.MdProps(dgpo)

	// @ process extension

	// @ hardfork_state

	tmpTrx := &prototype.TransactionWrapper{}
	tmpTrx.Receipt = &prototype.TransactionReceipt{}

	if skip&prototype.Skip_apply_transaction == 0 {

		for _, tw := range blk.Transactions {
			func() {
				trxContext := NewTrxContext(tmpTrx, c.db, c)
				defer func() {
					if err := recover(); err != nil {
						switch x := err.(type) {
						case *prototype.Exception:
							if x.ErrorType != prototype.StatusDeductGas {
								panic(err)
							}
						}
					}
					c.PayGas(trxContext)
					trxContext.Finalize()
					mustSuccess(tmpTrx.Receipt.Status == tw.Receipt.Status, "mismatched status", prototype.StatusErrorTrxApplyReceipt)
					mustSuccess(tmpTrx.Receipt.NetUsage == tw.Receipt.NetUsage, "mismatch net use", prototype.StatusErrorTrxApplyReceipt)
					mustSuccess(tmpTrx.Receipt.CpuUsage == tw.Receipt.CpuUsage, "mismatch cpu use", prototype.StatusErrorTrxApplyReceipt)
				}()
				tmpTrx.SigTrx = tw.SigTrx
				tmpTrx.Receipt.Status = prototype.StatusError
				c.applyTransaction(trxContext)
				//c.currentTrxInBlock++
			}()
		}
	}

	c.updateGlobalProperties(blk)
	//c.updateSigningWitness(blk)
	c.shuffle(blk)
	// @ update_last_irreversible_block
	c.createBlockSummary(blk)
	c.clearExpiredTransactions()
	// @ ...

	// @ notify_applied_block
}

func (c *TrxPool) PayGas(trxContext *TrxContext) (i interface{}) {
	i = nil
	defer func() {
		if err := recover(); err != nil {
			mustNoError(c.db.EndTransaction(false), "EndTransaction error", prototype.StatusErrorDbEndTrx)
			i = err
		}
	}()
	c.db.BeginTransaction()
	trxContext.DeductAllCpu()
	trxContext.DeductAllNet()
	mustNoError(c.db.EndTransaction(true), "EndTransaction error", prototype.StatusErrorDbEndTrx)
	return
}

func (c *TrxPool) ValidateAddress(name string, pubKey *prototype.PublicKeyType) bool {
	account := &prototype.AccountName{Value: name}
	witnessWrap := table.NewSoWitnessWrap(c.db, account)
	if !witnessWrap.CheckExist() {
		return false
	}
	dbPubKey := witnessWrap.GetSigningKey()
	if dbPubKey == nil {
		return false
	}

	return pubKey.Equal(dbPubKey)

	//authWrap := table.NewSoAccountAuthorityObjectWrap(c.db, account)
	//auth := authWrap.GetOwner()
	//if auth == nil {
	//	panic("no owner auth")
	//}
	//for _, k := range auth.KeyAuths {
	//	if pubKey.Equal(k.Key) {
	//		return true
	//	}
	//}
	//fmt.Println("ValidateAddress failed, ", name)
	//for _, k := range auth.KeyAuths {
	//	fmt.Println(k.Key.ToWIF())
	//}
	//fmt.Println("want ", pubKey.ToWIF())
	//return false
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
	pubKey, _ := prototype.PublicKeyFromWIF(constants.InitminerPubKey)
	name := &prototype.AccountName{Value: constants.COSInitMiner}
	newAccountWrap := table.NewSoAccountWrap(c.db, name)
	mustNoError(newAccountWrap.Create(func(tInfo *table.SoAccount) {
		tInfo.Name = name
		tInfo.CreatedTime = &prototype.TimePointSec{UtcSeconds: 0}
		tInfo.Balance = prototype.NewCoin(constants.COSInitSupply - 1000)
		tInfo.VestingShares = prototype.NewVest(1000)
		tInfo.LastPostTime = &prototype.TimePointSec{UtcSeconds: 0}
		tInfo.LastVoteTime = &prototype.TimePointSec{UtcSeconds: 0}
		tInfo.StakeVesting = prototype.NewVest(0)
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
		tInfo.CreatedTime = &prototype.TimePointSec{UtcSeconds: 0}
		tInfo.SigningKey = pubKey
		tInfo.LastWork = &prototype.Sha256{Hash: []byte{0}}
	}), "Witness Create Error", prototype.StatusErrorDbCreate)

	// create dynamic global properties
	dgpWrap := table.NewSoGlobalWrap(c.db, &constants.GlobalId)
	mustNoError(dgpWrap.Create(func(tInfo *table.SoGlobal) {
		tInfo.Id = constants.GlobalId
		tInfo.Props = &prototype.DynamicProperties{}
		tInfo.Props.CurrentWitness = name
		tInfo.Props.Time = &prototype.TimePointSec{UtcSeconds: constants.GenesisTime}
		tInfo.Props.HeadBlockId = &prototype.Sha256{Hash: make([]byte, 32)}
		// @ recent_slots_filled
		// @ participation_count
		tInfo.Props.CurrentSupply = prototype.NewCoin(constants.COSInitSupply - 1000)
		tInfo.Props.TotalCos = prototype.NewCoin(constants.COSInitSupply - 1000)
		tInfo.Props.MaximumBlockSize = constants.MaxBlockSize
		tInfo.Props.TotalVestingShares = prototype.NewVest(1000)
	}), "CreateDynamicGlobalProperties error", prototype.StatusErrorDbCreate)

	//create rewards keeper
	keeperWrap := table.NewSoRewardsKeeperWrap(c.db, &constants.GlobalId)
	rewards := make(map[string]*prototype.Vest)
	rewards["initminer"] = &prototype.Vest{Value: 0}
	mustNoError(keeperWrap.Create(func(tInfo *table.SoRewardsKeeper) {
		tInfo.Id = constants.GlobalId
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
	witnessScheduleWrap := table.NewSoWitnessScheduleObjectWrap(c.db, &constants.GlobalId)
	mustNoError(witnessScheduleWrap.Create(func(tInfo *table.SoWitnessScheduleObject) {
		tInfo.Id = constants.GlobalId
		tInfo.CurrentShuffledWitness = append(tInfo.CurrentShuffledWitness, constants.COSInitMiner)
	}), "CreateWitnessScheduleObject error", prototype.StatusErrorDbCreate)
}

func (c *TrxPool) TransferToVest(value *prototype.Coin) {
	c.modifyGlobalDynamicData(func(dgpo *prototype.DynamicProperties) {
		cos := dgpo.GetTotalCos()
		vest := dgpo.GetTotalVestingShares()
		addVest := value.ToVest()
		mustNoError(cos.Sub(value), "TotalCos overflow", prototype.StatusErrorTrxOverflow)
		dgpo.TotalCos = cos
		mustNoError(vest.Add(addVest), "TotalVestingShares overflow", prototype.StatusErrorTrxOverflow)
		dgpo.TotalVestingShares = vest
	})
}

func (c *TrxPool) TransferFromVest(value *prototype.Vest) {
	c.modifyGlobalDynamicData(func(dgpo *prototype.DynamicProperties) {
		cos := dgpo.GetTotalCos()
		vest := dgpo.GetTotalVestingShares()
		addCos := value.ToCoin()
		mustNoError(cos.Add(addCos), "TotalCos overflow", prototype.StatusErrorTrxOverflow)
		dgpo.TotalCos = cos
		mustNoError(vest.Sub(value), "TotalVestingShares overflow", prototype.StatusErrorTrxOverflow)
		dgpo.TotalVestingShares = vest
	})
}

func (c *TrxPool) validateBlockHeader(blk *prototype.SignedBlock) {
	headID := c.headBlockID()
	if !bytes.Equal(headID.Hash, blk.SignedHeader.Header.Previous.Hash) {
		e := &prototype.Exception{HelpString: "hash not equal", ErrorType: prototype.StatusErrorTrxBlockHeaderCheck}
		panic(e)
	}
	headTime := c.headBlockTime()
	if headTime.UtcSeconds >= blk.SignedHeader.Header.Timestamp.UtcSeconds {
		e := &prototype.Exception{HelpString: "block time is invalid", ErrorType: prototype.StatusErrorTrxBlockHeaderCheck}
		panic(e)
	}

	// witness sig check
	witnessName := blk.SignedHeader.Header.Witness
	witnessWrap := table.NewSoWitnessWrap(c.db, witnessName)
	pubKey := witnessWrap.GetSigningKey()
	res, err := blk.SignedHeader.ValidateSig(pubKey)
	if !res || err != nil {
		e := &prototype.Exception{ErrorString: err.Error(), HelpString: "block signature check error", ErrorType: prototype.StatusErrorTrxBlockHeaderCheck}
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
		genesisTime.UtcSeconds += slot * constants.BlockInterval
		return genesisTime
	}

	headBlockAbsSlot := c.headBlockTime().UtcSeconds / constants.BlockInterval
	slotTime := &prototype.TimePointSec{UtcSeconds: headBlockAbsSlot * constants.BlockInterval}

	slotTime.UtcSeconds += slot * constants.BlockInterval
	return slotTime
}

func (c *TrxPool) GetIncrementSlotAtTime(t *prototype.TimePointSec) uint32 {
	/*nextBlockSlotTime := c.GetSlotTime(1)
	if t.UtcSeconds < nextBlockSlotTime.UtcSeconds {
		return 0
	}
	return (t.UtcSeconds-nextBlockSlotTime.UtcSeconds)/constants.BlockInterval + 1*/
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
	dgpWrap := table.NewSoGlobalWrap(c.db, &constants.GlobalId)
	mustSuccess(dgpWrap.MdProps(dgpo), "update global data error", prototype.StatusErrorDbUpdate)
}

func (c *TrxPool) modifyGlobalDynamicData(f func(props *prototype.DynamicProperties)) {
	dgpWrap := table.NewSoGlobalWrap(c.db, &constants.GlobalId)
	props := dgpWrap.GetProps()

	f(props)

	mustSuccess(dgpWrap.MdProps(props), "", prototype.StatusErrorDbUpdate)
}

func (c *TrxPool) updateGlobalProperties(blk *prototype.SignedBlock) {
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
				if c.headBlockNum() - witnessWrap.GetLastConfirmedBlockNum() > constants.BlocksPerDay {
					emptyKey := &prototype.PublicKeyType{Data:[]byte{0}}
					witnessWrap.MdSigningKey(emptyKey)
					// @ push push_virtual_operation shutdown_witness_operation
				}
			}
		}*/

	// @ calculate participation

	id := blk.Id()
	blockID := &prototype.Sha256{Hash: id.Data[:]}

	c.modifyGlobalDynamicData(func(dgpo *prototype.DynamicProperties) {
		dgpo.HeadBlockNumber = blk.Id().BlockNum()
		dgpo.HeadBlockId = blockID
		dgpo.HeadBlockPrefix = binary.BigEndian.Uint32(id.Data[8:12])
		dgpo.Time = blk.SignedHeader.Header.Timestamp

		trxCount := len(blk.Transactions)
		dgpo.TotalTrxCnt += uint64(trxCount)
		dgpo.Tps = uint32(trxCount / constants.BlockInterval)

		if dgpo.MaxTps < dgpo.Tps {
			dgpo.MaxTps = dgpo.Tps
		}
	})

	c.noticer.Publish(constants.NoticeAddTrx, blk)
	// this check is useful ?
	//mustSuccess(dgpo.GetHeadBlockNumber()-dgpo.GetIrreversibleBlockNum() < constants.MAX_UNDO_HISTORY, "The database does not have enough undo history to support a blockchain with so many missed blocks.",prototype.StatusErrorTrxMaxUndo)
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
	mustSuccess(blockSummaryWrap.CheckExist(), "can not get block summary object", prototype.StatusErrorDbExist)
	blockIDArray := blk.Id().Data
	blockID := &prototype.Sha256{Hash: blockIDArray[:]}
	mustSuccess(blockSummaryWrap.MdBlockId(blockID), "update block summary object error", prototype.StatusErrorDbUpdate)
}

func (c *TrxPool) clearExpiredTransactions() {
	sortWrap := table.STransactionObjectExpirationWrap{Dba: c.db}
	sortWrap.ForEachByOrder(nil, nil, nil, nil,
		func(mVal *prototype.Sha256, sVal *prototype.TimePointSec, idx uint32) bool {
			if sVal != nil {
				headTime := c.headBlockTime().UtcSeconds
				if headTime > sVal.UtcSeconds {
					// delete trx ...
					k := mVal
					objWrap := table.NewSoTransactionObjectWrap(c.db, k)
					mustSuccess(objWrap.RemoveTransactionObject(), "RemoveTransactionObject error", prototype.StatusErrorDbDelete)
				}
				return true
			}
			return false
		})
}

func (c *TrxPool) GetWitnessTopN(n uint32) []string {
	var ret []string
	revList := table.SWitnessVoteCountWrap{Dba: c.db}
	revList.ForEachByRevOrder(nil, nil, nil, nil, func(mVal *prototype.AccountName, sVal *uint64, idx uint32) bool {
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
	witnessScheduleWrap := table.NewSoWitnessScheduleObjectWrap(c.db, &constants.GlobalId)
	mustSuccess(witnessScheduleWrap.MdCurrentShuffledWitness(names), "SetWitness error", prototype.StatusErrorDbUpdate)
}

func (c *TrxPool) GetShuffledWitness() []string {
	witnessScheduleWrap := table.NewSoWitnessScheduleObjectWrap(c.db, &constants.GlobalId)
	return witnessScheduleWrap.GetCurrentShuffledWitness()
}

func (c *TrxPool) AddWeightedVP(value uint64) {
	dgpo := c.GetProps()
	dgpo.WeightedVps += value
	dgpWrap := table.NewSoGlobalWrap(c.db, &constants.GlobalId)
	dgpWrap.MdProps(dgpo)
}

func (c *TrxPool) PopBlock(num uint64) {
	// undo pending trx
	c.ClearPending()
	/*if c.havePendingTransaction {
		mustNoError(c.db.EndTransaction(false), "EndTransaction error")
		c.havePendingTransaction = false
		//c.log.Debug("@@@@@@ PopBlock havePendingTransaction=false")
	}*/
	// get reversion
	//rev := c.getReversion(num)
	//mustNoError(c.db.RevertToRevision(rev), fmt.Sprintf("RebaseToRevision error: tag:%d, reversion:%d", num, rev))
	err := c.iceberg.RevertBlock(num)
	mustSuccess(err == nil, fmt.Sprintf("revert block %d, error: %v", num, err), prototype.StatusErrorDbTag)
}

func (c *TrxPool) Commit(num uint64) {
	// this block can not be revert over, so it's irreversible
	err := c.iceberg.FinalizeBlock(num)
	mustSuccess(err == nil, fmt.Sprintf("commit block: %d, error is %v", num, err), prototype.StatusErrorDbTag)
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
	if bytes.Equal(dbPubKey.Data, result.Data) {
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

func (c *TrxPool) GetCommitBlockNum() (uint64, error) {
	return c.iceberg.LastFinalizedBlock()
}

//Sync committed blocks to squash db when node reStart
func (c *TrxPool) SyncCommittedBlockToDB(blk common.ISignedBlock) (err error) {
	defer func() {
		if r := recover(); r != nil {
			desc := fmt.Sprintf("[Sync commit]:Faile to commit,the error is %v", err)
			err = errors.New(desc)
		}
	}()
	if blk == nil {
		return errors.New("[Sync commit]:Fail to sync commit nil block")
	}
	cmtNum, err := c.GetCommitBlockNum()
	if err != nil {
		return err
	}
	num := blk.Id().BlockNum()
	if num <= cmtNum {
		desc := fmt.Sprintf("[Sync commit]: the block of num %d has already commit,current "+
			"commit num is %d", num, cmtNum)
		err = errors.New(desc)
		return err
	}
	c.log.Debugf("[Reload commit] :sync lost commit block which num is %d", num)
	pErr := c.PushBlock(blk.(*prototype.SignedBlock), prototype.Skip_nothing)
	if pErr != nil {
		desc := fmt.Sprintf("[Sync commit]: push the block which num is %v fail,error is %s", num, pErr)
		err = errors.New(desc)
		return err
	}
	c.Commit(num)
	return err
}

//Sync pushed blocks to squash db when node reStart
func (c *TrxPool) SyncPushedBlocksToDB(blkList []common.ISignedBlock) (err error) {
	defer func() {
		if r := recover(); r != nil {
			desc := fmt.Sprintf("[Sync pushed]:Faile to push block,the error is %v", err)
			err = errors.New(desc)
		}
	}()
	if blkList != nil {
		cmtNum, err := c.GetCommitBlockNum()
		if err != nil {
			return err
		}
		for i := range blkList {
			blk := blkList[i]
			num := blk.Id().BlockNum()
			if cmtNum >= num {
				desc := fmt.Sprintf("[sync pushed]: the block num %v is not greater than "+
					"the latest commit block num %v", num, cmtNum)
				return errors.New(desc)
			}
			c.log.Debugf("[sync pushed]: sync pushed block,blockNum is: "+
				"%v", blk.(*prototype.SignedBlock).Id().BlockNum())
			err := c.PushBlock(blk.(*prototype.SignedBlock), prototype.Skip_nothing)
			if err != nil {
				desc := fmt.Sprintf("[sync pushed]: push the block which num is %v fail,error is %s", i, err)
				return errors.New(desc)
			}
		}
	}
	return err
}

func (c *TrxPool) GetRemainStamina(name string) uint64 {
	wraper := table.NewSoGlobalWrap(c.db, &constants.GlobalId)
	gp := wraper.GetProps()
	return c.resourceLimiter.GetStakeLeft(name, gp.HeadBlockNumber)
}

func (c *TrxPool) GetRemainFreeStamina(name string) uint64 {
	wraper := table.NewSoGlobalWrap(c.db, &constants.GlobalId)
	gp := wraper.GetProps()
	return c.resourceLimiter.GetFreeLeft(name, gp.HeadBlockNumber)
}

func (c *TrxPool) GetStaminaMax(name string) uint64 {
	return c.resourceLimiter.GetCapacity(name)
}

func (c *TrxPool) GetStaminaFreeMax() uint64 {
	return c.resourceLimiter.GetCapacityFree()
}

func (c *TrxPool) GetAllRemainStamina(name string) uint64 {
	return c.GetRemainStamina(name) + c.GetRemainFreeStamina(name)
}

func (c *TrxPool) GetAllStaminaMax(name string) uint64 {
	return c.GetStaminaMax(name) + c.GetStaminaFreeMax()
}
