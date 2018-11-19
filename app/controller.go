package app

import (
	"bytes"
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/common/eventloop"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/prototype"
	"github.com/golang/protobuf/proto"
	"time"
)

type skipFlag uint32

const (
	skip_nothing                skipFlag = 0
	skip_transaction_signatures skipFlag = 1 << 0
	skip_apply_transaction      skipFlag = 1 << 1
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
	skip    skipFlag

	_pending_tx        []*prototype.TransactionWrapper
	_isProducing       bool
	_currentTrxId      *prototype.Sha256
	_current_op_in_trx uint16
	_currentBlockNum 	uint64
	_current_trx_in_block int16
}

func (c *Controller) getDb() (iservices.IDatabaseService,error) {
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
	db,err := c.getDb()
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
	var i int32 = 0
	dgpWrap := table.NewSoDynamicGlobalPropertiesWrap(c.db,&i)
	if !dgpWrap.CheckExist() {
		c.initGenesis()
	}
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
		c.setProducing(false)
		c.skip = oldSkip
	}()

	// check maximum_block_size
	var i int32 = 0
	dgpWrap := table.NewSoDynamicGlobalPropertiesWrap(c.db,&i)
	if  proto.Size(trx) > int(dgpWrap.GetMaximumBlockSize() - 256) {
		panic("transaction is too large")
	}

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
	if len(c._pending_tx) == 0 {
		c.db.BeginTransaction()
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
	//c.NotifyTrxPending(trx)
	return trxWrp.Invoice
}

func (c *Controller) PushBlock(blk *prototype.SignedBlock) {

}

func (c *Controller) GenerateBlock(accountName string, timestamp uint32,
					 prev common.BlockID) *prototype.SignedBlock {
	return nil
}

func (c *Controller) NotifyOpPostExecute(on *prototype.OperationNotification) {
	c.noticer.Publish(constants.NOTICE_OP_POST, on)
}

func (c *Controller) NotifyOpPreExecute(on *prototype.OperationNotification) {
	c.noticer.Publish(constants.NOTICE_OP_PRE, on)
}

func (c *Controller) NotifyTrxPreExecute(trx *prototype.SignedTransaction) {
	c.noticer.Publish(constants.NOTICE_TRX_PRE, trx)
}

func (c *Controller) NotifyTrxPostExecute(trx *prototype.SignedTransaction) {
	c.noticer.Publish(constants.NOTICE_TRX_POST, trx)
}

func (c *Controller) NotifyTrxPending(trx *prototype.SignedTransaction) {
	c.noticer.Publish(constants.NOTICE_TRX_PENDING, trx)
}

func (c *Controller) NotifyBlockApply(block *prototype.SignedBlock) {
	c.noticer.Publish(constants.NOTICE_BLOCK_APPLY, block)
}

// calculate reward for creator and witness
func (c *Controller) processBlock() {
}

func (c *Controller) _applyTransaction(trxWrp *prototype.TransactionWrapper) {
	defer func() {
		if err := recover(); err != nil {
			trxWrp.Invoice.Status = 500
			panic("_applyTransaction failed")
		} else {
			trxWrp.Invoice.Status = 200
			return
		}
	}()

	trx := trxWrp.SigTrx
	var err error
	c._currentTrxId, err = trx.Id()
	if err != nil {
		panic("get trx id failed")
	}

	trx.Validate()

	// trx duplicate check
	transactionObjWrap := table.NewSoTransactionObjectWrap(c.db, c._currentTrxId)
	if transactionObjWrap.CheckExist() {
		panic("Duplicate transaction check failed")
	}

	if c.skip&skip_transaction_signatures == 0 {
		postingGetter := func(name string) *prototype.Authority {
			account := &prototype.AccountName{Value:name}
			authWrap := table.NewSoAccountAuthorityObjectWrap(c.db,account)
			auth := authWrap.GetPosting()
			if auth == nil {
				panic("no posting auth")
			}
			return auth
		}
		activeGetter := func(name string) *prototype.Authority {
			account := &prototype.AccountName{Value:name}
			authWrap := table.NewSoAccountAuthorityObjectWrap(c.db,account)
			auth := authWrap.GetActive()
			if auth == nil {
				panic("no posting auth")
			}
			return auth
		}
		ownerGetter := func(name string) *prototype.Authority {
			account := &prototype.AccountName{Value:name}
			authWrap := table.NewSoAccountAuthorityObjectWrap(c.db,account)
			auth := authWrap.GetOwner()
			if auth == nil {
				panic("no posting auth")
			}
			return auth
		}

		tmpChainId := prototype.ChainId{Value: 0}
		trx.VerifyAuthority(tmpChainId, 2,postingGetter,activeGetter,ownerGetter)
		// @ check_admin
	}

	// TaPos and expired check
	var i int32 = 0
	dgpWrap := table.NewSoDynamicGlobalPropertiesWrap(c.db,&i)
	blockNum := dgpWrap.GetHeadBlockNumber()
	if blockNum > 0 {
		uniWrap := table.UniBlockSummaryObjectIdWrap{}
		idWrap := uniWrap.UniQueryId(&trx.Trx.RefBlockNum)
		if !idWrap.CheckExist() {
			panic("no refBlockNum founded")
		} else {
			blockId := idWrap.GetBlockId()
			summaryId := uint32(blockId.Hash[1])
			if trx.Trx.RefBlockPrefix != summaryId {
				panic("transaction tapos failed")
			}
		}
		// get head time
		if trx.Trx.Expiration.UtcSeconds > uint32(time.Now().Second()+30) {
			panic("transaction expiration too long")
		}
		if uint32(time.Now().Second()) > trx.Trx.Expiration.UtcSeconds {
			panic("transaction has expired")
		}
	}

	// insert trx into DB unique table
	obj := &table.SoTransactionObject{}
	obj.TrxId = c._currentTrxId
	obj.Expiration = &prototype.TimePointSec{UtcSeconds: 100}
	if !transactionObjWrap.CreateTransactionObject(obj) {
		panic("create transactionObject failed")
	}
	// @ not use yet
	//c.NotifyTrxPreExecute(trx)

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
	//n := &prototype.OperationNotification{Op: op}
//	c.NotifyOpPreExecute(n)
	eva := c.getEvaluator(op)
	eva.Apply(op)
	// @ not use yet
//	c.NotifyOpPostExecute(n)
}

func (c *Controller) getEvaluator(op *prototype.Operation) BaseEvaluator {
	ctx := &ApplyContext{ db:c.db, control:c}
	switch op.Op.(type) {
	case *prototype.Operation_Op1:
		eva := &AccountCreateEvaluator{ ctx:ctx, op: op.GetOp1() }
		return BaseEvaluator(eva)
	case *prototype.Operation_Op2:
		eva := &TransferEvaluator{ ctx:ctx, op: op.GetOp2() }
		return BaseEvaluator(eva)
	default:
		panic("no matchable evaluator")
	}
}

func (c *Controller) applyBlock(blk *prototype.SignedBlock) {
	oldFlag := c.skip
	defer func() {
		c.skip = oldFlag
	}()

	c._applyBlock(blk)

	// @ tps update
}

func (c *Controller) _applyBlock(blk *prototype.SignedBlock) {
	nextBlockNum := blk.Id().BlockNum()

	merkleRoot := blk.CalculateMerkleRoot()
	if !bytes.Equal(merkleRoot.Data[:], blk.SignedHeader.Header.TransactionMerkleRoot.Hash) {
		panic("Merkle check failed")
	}

	// @ validate_block_header

	c._currentBlockNum = nextBlockNum
	c._current_trx_in_block = 0

	var i int32 = 0
	dgpWrap := table.NewSoDynamicGlobalPropertiesWrap(c.db,&i)
	blockSize := proto.Size(blk)
	if uint32(blockSize) > dgpWrap.GetMaximumBlockSize() {
		panic("Block size is too big")
	}
	if uint32(blockSize) < constants.MIN_BLOCK_SIZE {
		// elog("Block size is too small")
	}

	w := prototype.AccountName{Value:blk.SignedHeader.Header.Witness}
	dgpWrap.MdCurrentWitness(w)

	// @ process extension

	// @ hardfork_state

	trxWrp := &prototype.TransactionWrapper{}
	trxWrp.Invoice = &prototype.TransactionInvoice{}

	for _,tw := range blk.Transactions {
		trxWrp.SigTrx = tw.SigTrx
		trxWrp.Invoice.Status = 200
		c._applyTransaction(trxWrp)
		if trxWrp.Invoice.Status != tw.Invoice.Status {
			panic("mismatched invoice")
		}
		c._current_trx_in_block++
	}

	// update xxx ...
}

func (c *Controller) initGenesis() {

	// create initminer
	pubKey , _ := prototype.PublicKeyFromWIF(constants.INITMINER_PUBKEY)
	name := &prototype.AccountName{Value:constants.INIT_MINER_NAME}
	newAccountWrap := table.NewSoAccountWrap(c.db,name)
	newAccount := &table.SoAccount{}
	newAccount.Name = name
	newAccount.PubKey = pubKey
	newAccount.CreatedTime = &prototype.TimePointSec{UtcSeconds:0}
	cos := prototype.MakeCoin(constants.INIT_SUPPLY)
	vest := prototype.MakeVest(0)
	newAccount.Balance = cos
	newAccount.VestingShares = vest
	if !newAccountWrap.CreateAccount(newAccount) {
		panic("CreateAccount error")
	}

	// create account authority
	authorityWrap := table.NewSoAccountAuthorityObjectWrap(c.db,name)
	authority := &table.SoAccountAuthorityObject{}
	authority.Account = name

	ownerAuth := &prototype.Authority{
		WeightThreshold: 1,
		KeyAuths: []*prototype.KvKeyAuth{
			&prototype.KvKeyAuth{
				Key: pubKey,
				Weight: 1,
			},
		},
	}
	authority.Posting = ownerAuth
	authority.Active = ownerAuth
	authority.Owner = ownerAuth
	if !authorityWrap.CreateAccountAuthorityObject(authority) {
		panic("CreateAccountAuthorityObject error ")
	}
	// @ create witness_object

	// create dynamic global properties
	var i int32 = 0
	dgpWrap := table.NewSoDynamicGlobalPropertiesWrap(c.db,&i)
	dgp := &table.SoDynamicGlobalProperties{}
	dgp.CurrentWitness = name
	dgp.Time = &prototype.TimePointSec{UtcSeconds:constants.GENESIS_TIME}
	// @ recent_slots_filled
	// @ participation_count
	dgp.CurrentSupply = cos
	dgp.TotalCos = cos
	dgp.MaximumBlockSize = constants.MAX_BLOCK_SIZE
	dgp.TotalVestingShares = prototype.MakeVest(0)
	if !dgpWrap.CreateDynamicGlobalProperties(dgp) {
		panic("CreateDynamicGlobalProperties error")
	}

	// create block summary
	for i := uint32(0); i < 0x10000; i++ {
		wrap := table.NewSoBlockSummaryObjectWrap(c.db, &i)
		obj := &table.SoBlockSummaryObject{}
		obj.Id = i
		if !wrap.CreateBlockSummaryObject(obj) {
			panic("CreateBlockSummaryObject error")
		}
	}
}

func (c *Controller) CreateVesting(accountName *prototype.AccountName, cos *prototype.Coin) *prototype.Vest {

	newVesting := prototype.CosToVesting(cos)
	creatorWrap := table.NewSoAccountWrap(c.db,accountName)
	oldVesting := creatorWrap.GetVestingShares()
	oldVesting.Value += newVesting.Value
	creatorWrap.MdVestingShares(oldVesting)

	var i int32 = 0
	dgpWrap := table.NewSoDynamicGlobalPropertiesWrap(c.db,&i)
	originTotal := dgpWrap.GetTotalVestingShares()
	originTotal.Value += newVesting.Value
	dgpWrap.MdTotalVestingShares(originTotal)
	return newVesting
}

func (c *Controller) SubBalance(accountName *prototype.AccountName, cos *prototype.Coin) {
	accountWrap := table.NewSoAccountWrap(c.db,accountName)
	originBalance := accountWrap.GetBalance()
	originBalance.Value -= cos.Value
	accountWrap.MdBalance(originBalance)

	// dynamic glaobal properties
	var i int32 = 0
	dgpWrap := table.NewSoDynamicGlobalPropertiesWrap(c.db,&i)
	originTotal := dgpWrap.GetTotalCos()
	originTotal.Value -= cos.Value
	dgpWrap.MdTotalCos(originTotal)
}

func (c *Controller) AddBalance(accountName *prototype.AccountName, cos *prototype.Coin) {
	accountWrap := table.NewSoAccountWrap(c.db,accountName)
	originBalance := accountWrap.GetBalance()
	originBalance.Value += cos.Value
	accountWrap.MdBalance(originBalance)

	// dynamic glaobal properties
	var i int32 = 0
	dgpWrap := table.NewSoDynamicGlobalPropertiesWrap(c.db,&i)
	originTotal := dgpWrap.GetTotalCos()
	originTotal.Value += cos.Value
	dgpWrap.MdTotalCos(originTotal)
}


func (c *Controller) validateBlockHeader(blk *prototype.SignedBlock) {
	headID := c.headBlockID()
	if !bytes.Equal(headID.Hash,blk.SignedHeader.Header.Previous.Hash) {
		panic("hash not equal")
	}
	headTime := c.headBlockTime()
	if headTime.UtcSeconds >= blk.SignedHeader.Header.Timestamp.UtcSeconds {
		panic("block time is invalid")
	}


}


func (c *Controller) headBlockID() *prototype.Sha256 {
	var i int32 = 0
	dgpWrap := table.NewSoDynamicGlobalPropertiesWrap(c.db,&i)
	headID := dgpWrap.GetHeadBlockId()
	return headID
}

func (c *Controller) headBlockTime() *prototype.TimePointSec {
	var i int32 = 0
	dgpWrap := table.NewSoDynamicGlobalPropertiesWrap(c.db,&i)
	return dgpWrap.GetTime()
}