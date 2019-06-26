package app

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common"
	selfmath "github.com/coschain/contentos-go/common/math"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/common/eventloop"
	"github.com/coschain/contentos-go/utils"
	"math"
	"math/big"
	"sort"

	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/prototype"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"time"
)

var (
	SingleId int32 = 1
)

type TrxPool struct {
	iservices.ITrxPool

	ctx    *node.ServiceContext
	evLoop *eventloop.EventLoop
	db      iservices.IDatabaseService
	log     *logrus.Logger
	noticer EventBus.Bus
	shuffle                common.ShuffleFunc

	iceberg   *BlockIceberg
	economist *Economist
	stateObserver *StateObserver
	tm *TrxMgr

	resourceLimiter utils.IResourceLimiter
	enableBAH bool
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
	return &TrxPool{ctx: ctx, log: lg, enableBAH:false}, nil
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

		mustNoError(c.db.DeleteAll(), "truncate database error")

		//c.log.Info("start initGenesis")
		c.initGenesis()

		mustNoError(c.db.TagRevision(c.db.GetRevision(), GENESIS_TAG), "genesis tagging failed")
		//c.log.Info("finish initGenesis")
	}
	c.iceberg = NewBlockIceberg(c.db, c.log, c.enableBAH)
	c.economist = NewEconomist(c.db, c.noticer, c.log)
	c.stateObserver = NewStateObserver(c.noticer, c.log)

	commit, _ := c.iceberg.LastFinalizedBlock()
	latest, _, _ := c.iceberg.LatestBlock()
	c.tm = NewTrxMgr(c.db, c.log, latest, commit)
	c.resourceLimiter = utils.NewResourceLimiter()
}

func (c *TrxPool) Stop() error {
	return nil
}

func (c *TrxPool) PushTrxToPending(trx *prototype.SignedTransaction) (err error) {
	return c.tm.AddTrx(trx, nil)
}

func (c *TrxPool) PushTrx(trx *prototype.SignedTransaction) (invoice *prototype.TransactionReceiptWithInfo) {
	rc := make(chan *prototype.TransactionReceiptWithInfo)
	_ = c.tm.AddTrx(trx, func(result *prototype.TransactionWrapperWithInfo) {
		rc <- result.Receipt
	})
	return <-rc
}

func (c *TrxPool) EstimateStamina(trx *prototype.SignedTransaction) (invoice *prototype.TransactionReceiptWithInfo) {
	c.db.Lock()
	defer c.db.Unlock()
	entry := NewTrxMgrEntry(trx, nil)
	invoice = &prototype.TransactionReceiptWithInfo{}
	if err := entry.InitCheck(); err != nil {
		invoice.ErrorInfo = err.Error()
		return
	}
	db := c.db.NewPatch()

	defer func() {
		// recover from panic and return an error
		if e := recover(); e != nil {
			invoice.ErrorInfo = fmt.Sprintf("%v", e)
		}
		invoice.NetUsage = entry.GetTrxResult().Receipt.NetUsage
		invoice.CpuUsage = entry.GetTrxResult().Receipt.CpuUsage
	}()
	c.applyTransactionOnDb(db,entry)
	return
}

func (c *TrxPool) GetProps() *prototype.DynamicProperties {
	dgpWrap := table.NewSoGlobalWrap(c.db, &SingleId)
	return dgpWrap.GetProps()
}

func (c *TrxPool) PushBlock(blk *prototype.SignedBlock, skip prototype.SkipFlag) (err error) {
	c.db.Lock()
	defer c.db.Unlock()

	return c.pushBlockNoLock(blk, skip)
}

func (c *TrxPool) pushBlockNoLock(blk *prototype.SignedBlock, skip prototype.SkipFlag) (err error) {

	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {
			case error:
				err = x
			case string:
				err = errors.New(x)
			default:
				err = errors.New("unknown panic type")
			}
			// undo changes
			_ = c.iceberg.EndBlock(false)
			c.stateObserver.EndBlock("")
			c.log.Debug("ICEBERG: EndBlock FALSE")
			c.log.Errorf("push block fail,the error is %v,the block num is %v", r, blk.Id().BlockNum())
		}
	}()

	if skip&prototype.Skip_apply_transaction == 0 {
		blkNum := blk.Id().BlockNum()
		c.log.Debugf("ICEBERG: BeginBlock %d", blkNum)
		_ = c.iceberg.BeginBlock(blkNum)
		c.stateObserver.BeginBlock(blkNum)
		c.applyBlock(blk, skip)
		data := blk.Id().Data
		c.stateObserver.EndBlock(hex.EncodeToString(data[:]))
		c.log.Debug("ICEBERG: EndBlock TRUE")
		_ = c.iceberg.EndBlock(true)
	} else {
		// we have do a BeginTransaction at GenerateBlock
		c.applyBlock(blk, skip)
		c.log.Debug("ICEBERG: EndBlock TRUE")
		data := blk.Id().Data
		c.stateObserver.EndBlock(hex.EncodeToString(data[:]))
		_ = c.iceberg.EndBlock(true)
	}

	return err
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

	s := time.Now()
	blockChan := make(chan interface{})

	go func() {
		defer func() {
			c.log.Debug("[trxpool] GenerateAndApplyBlock cost: ", time.Now().Sub(s))
		}()

		c.db.Lock()
		defer c.db.Unlock()

		newBlock, err := c.generateBlockNoLock(witness, pre, timestamp, priKey, skip, s)
		if err != nil {
			blockChan <- err
		} else {
			blockChan <- newBlock
		}
		close(blockChan)

		if err == nil {
			if err = c.pushBlockNoLock(newBlock, skip|prototype.Skip_apply_transaction|prototype.Skip_block_check); err != nil {
				c.log.Errorf("pushBlockNoLock failed: %v", err)
			}
		}
	}()

	blockOrError := <- blockChan
	if b, ok := blockOrError.(*prototype.SignedBlock); ok {
		return b, nil
	} else {
		return nil, blockOrError.(error)
	}
}

func (c *TrxPool) GenerateBlock(witness string, pre *prototype.Sha256, timestamp uint32,
	priKey *prototype.PrivateKeyType, skip prototype.SkipFlag) (b *prototype.SignedBlock, e error) {

	entryTime := time.Now()
	c.db.Lock()
	defer c.db.Unlock()

	return c.generateBlockNoLock(witness, pre, timestamp, priKey, skip, entryTime)
}

func (c *TrxPool) generateBlockNoLock(witness string, pre *prototype.Sha256, timestamp uint32,
	priKey *prototype.PrivateKeyType, skip prototype.SkipFlag, entryTime time.Time) (b *prototype.SignedBlock, e error) {

	const (
		maxTimeout = 700 * time.Millisecond
		minTimeout = 100 * time.Millisecond
	)

	timing := common.NewTiming()
	timing.Begin()

	defer func() {
		if err := recover(); err != nil {
			c.log.Debug("ICEBERG: EndBlock FALSE")
			_ = c.iceberg.EndBlock(false)
			c.stateObserver.EndBlock("")

			b, e = nil, fmt.Errorf("%v", err)
		}
	}()

	pubkey, err := priKey.PubKey()
	mustNoError(err, "get public key error")

	witnessWrap := table.NewSoWitnessWrap(c.db, &prototype.AccountName{Value: witness})
	mustSuccess(bytes.Equal(witnessWrap.GetSigningKey().Data[:], pubkey.Data[:]), "public key not equal")

	// @ signHeader size is zero, must have some content
	//signHeader := &prototype.SignedBlockHeader{}
	//emptyHeader(signHeader)

	dgpWrap := table.NewSoGlobalWrap(c.db, &SingleId)
	maxBlockSize := dgpWrap.GetProps().MaximumBlockSize

	signBlock := &prototype.SignedBlock{}
	signBlock.SignedHeader = &prototype.SignedBlockHeader{}
	signBlock.SignedHeader.Header = &prototype.BlockHeader{}

	blkNum := c.headBlockNum() + 1
	c.log.Debugf("ICEBERG: BeginBlock %d", blkNum)
	_ = c.iceberg.BeginBlock(blkNum)
	c.stateObserver.BeginBlock(blkNum)

	timeOut := maxTimeout - time.Since(entryTime)
	if timeOut < minTimeout {
		timeOut = minTimeout
	}
	isFinish := false
	time.AfterFunc(timeOut, func() {
		isFinish = true
	})

	const batchCount = 64
	ma := NewMultiTrxsApplier(c.db, c.applyTransactionOnDb)

	timing.Mark()

	applyTime := int64(0)
	sizeLimit := int(maxBlockSize)
	var failedTrx []*TrxEntry
	for {
		if isFinish {
			c.log.Warn("[trxpool] Generate block timeout, total pending: ", c.tm.WaitingCount() )
			break
		}
		trxs := c.tm.FetchTrx(timestamp, batchCount, sizeLimit)
		t00 := time.Now()
		ma.Apply(trxs)
		applyTime += int64(time.Now().Sub(t00))
		for _, entry := range trxs {
			result := entry.GetTrxResult()
			if result.Receipt.Status == prototype.StatusError {
				failedTrx = append(failedTrx, entry)
			} else {
				sizeLimit -= entry.GetTrxSize()
				signBlock.Transactions = append(signBlock.Transactions, result.ToWrapper())
			}
		}
		if sizeLimit <= 0 {
			c.log.Warnf("[trxpool] postponed %d trx due to max block size", c.tm.WaitingCount())
			break
		}
		if len(trxs) < batchCount {
			break
		}
	}

	timing.SetPartial(time.Duration(applyTime))
	timing.Mark()

	signBlock.SignedHeader.Header.Previous = pre
	signBlock.SignedHeader.Header.PrevApplyHash = c.iceberg.LatestBlockApplyHash()
	signBlock.SignedHeader.Header.Timestamp = &prototype.TimePointSec{UtcSeconds: timestamp}
	id := signBlock.CalculateMerkleRoot()
	signBlock.SignedHeader.Header.TransactionMerkleRoot = &prototype.Sha256{Hash: id.Data[:]}
	signBlock.SignedHeader.Header.Witness = &prototype.AccountName{Value: witness}
	signBlock.SignedHeader.WitnessSignature = &prototype.SignatureType{}
	_ = signBlock.SignedHeader.Sign(priKey)

	if len(failedTrx) > 0 {
		c.tm.ReturnTrx(failedTrx...)
	}

	timing.Mark()

	c.updateGlobalProperties(signBlock)

	timing.Mark()

	ret, bpNameList := c.shuffle(signBlock)
	if ret {
		c.updateGlobalWitnessBoot(bpNameList)
		c.updateGlobalResourceParam(bpNameList)
		c.deleteUnusedBp(bpNameList)
	}

	timing.Mark()

	c.updateAvgTps(signBlock)

	timing.End()

	c.log.Debugf("GENBLOCK %d: %s, timeout=%v, pending=%d, failed=%d, inblk=%d",
		signBlock.Id().BlockNum(), timing.String(), timeOut, c.tm.WaitingCount(), len(failedTrx), len(signBlock.Transactions))

	b, e = signBlock, nil
	return
}

func (c *TrxPool) notifyBlockApply(block *prototype.SignedBlock) {
	timing := common.NewTiming()
	timing.Begin()
	blockNum := block.Id().BlockNum()
	for trxIdx, trx := range block.Transactions {
		for opIdx, op := range trx.SigTrx.Trx.Operations {
			c.noticer.Publish(constants.NoticeOpPost,
				&prototype.OperationNotification{
					Trx_status: trx.Receipt.Status,
					Block: blockNum,
					Trx_in_block: uint64(trxIdx),
					Op_in_trx: uint64(opIdx),
					Op: op,
				})
		}
		c.noticer.Publish(constants.NoticeTrxPost, trx.SigTrx)
	}
	timing.Mark()
	c.noticer.Publish(constants.NoticeBlockApplied, block)
	timing.End()
	c.log.Debugf("NOTIFYBLOCK %d: %s, #tx=%d", blockNum, timing.String(), len(block.Transactions))
}

func (c *TrxPool) notifyTrxApplyResult(trx *prototype.SignedTransaction, res bool,
	receipt *prototype.TransactionReceiptWithInfo) {
	c.noticer.Publish(constants.NoticeTrxApplied, trx, receipt)
}

func (c *TrxPool) applyTransactionOnDb(db iservices.IDatabasePatch, entry *TrxEntry) {
	result := entry.GetTrxResult()
	receipt, sigTrx := result.GetReceipt(), result.GetSigTrx()

	trxDB := db.NewPatch()

	trxObserver := c.stateObserver.NewTrxObserver()
	cid := prototype.ChainId{Value: 0}
	trxHash, _ := sigTrx.GetTrxHash(cid)
	trxObserver.BeginTrx(hex.EncodeToString(trxHash))
	trxContext := NewTrxContext(result, trxDB, entry.GetTrxSigner(), c, trxObserver)

	defer func() {
		useGas := trxContext.HasGasFee()

		if err := recover(); err != nil {
			receipt.ErrorInfo = fmt.Sprintf("applyTransaction failed : %v", err)
			trxObserver.EndTrx(false)
			if useGas && constants.EnableResourceControl {
				receipt.Status = prototype.StatusDeductStamina
				c.notifyTrxApplyResult(sigTrx, true, receipt)
			} else {
				receipt.Status = prototype.StatusError
				c.notifyTrxApplyResult(sigTrx, false, receipt)
				panic(receipt.ErrorInfo)
			}
		} else {
			// commit changes to db
			_ = trxDB.Apply()
			receipt.Status = prototype.StatusSuccess
			trxObserver.EndTrx(true)
			c.notifyTrxApplyResult(sigTrx, true, receipt)
		}
		c.PayGas(db,trxContext)
	}()

	trxContext.CheckNet(trxDB, uint64(proto.Size(sigTrx)))

	for _, op := range sigTrx.Trx.Operations {
		trxContext.StartNextOp()
		c.applyOperation(trxContext, op)
	}
}

func (c *TrxPool) PayGas(db iservices.IDatabaseRW, trxContext *TrxContext) {
	trxContext.DeductAllCpu(db)
	trxContext.DeductAllNet(db)
	trxContext.Finalize()
	return
}

func (c *TrxPool) applyOperation(trxCtx *TrxContext, op *prototype.Operation) {
	eva := c.getEvaluator(trxCtx, op)
	eva.Apply()
}

func (c *TrxPool) getEvaluator(trxCtx *TrxContext, op *prototype.Operation) BaseEvaluator {
	return GetBaseEvaluator(trxCtx, op)
}

func (c *TrxPool) applyBlock(blk *prototype.SignedBlock, skip prototype.SkipFlag) {
	blockNum := blk.Id().BlockNum()

	if skip & prototype.Skip_block_check == 0 {
		merkleRoot := blk.CalculateMerkleRoot()
		mustSuccess(bytes.Equal(merkleRoot.Data[:], blk.SignedHeader.Header.TransactionMerkleRoot.Hash), "Merkle check failed")

		// validate_block_header
		c.validateBlockHeader(blk)

		blockSize := proto.Size(blk)
		mustSuccess(uint32(blockSize) <= c.GetProps().GetMaximumBlockSize(), "Block size is too big")

		if uint32(blockSize) < constants.MinBlockSize {
		}
	}

	if skip&prototype.Skip_apply_transaction == 0 {
		pushTiming := common.NewTiming()
		pushTiming.Begin()

		entries, err := c.tm.CheckBlockTrxs(blk)
		mustNoError(err, "block trxs check failed")

		applyTime := int64(0)
		ma := NewMultiTrxsApplier(c.db, c.applyTransactionOnDb)
		batchCount := 64
		totalCount := len(entries)
		for i := 0; i < totalCount; i += batchCount {
			d := totalCount - i
			if d > batchCount {
				d = batchCount
			}
			t00 := time.Now()
			ma.Apply(entries[i:i+d])
			applyTime += int64(time.Now().Sub(t00))
			invoiceOK := true
			for j := 0; j < d; j++ {
				trxIdx := i + j
				expect := blk.Transactions[trxIdx].Receipt
				actual := entries[trxIdx].GetTrxResult().Receipt

				if actual.Status != expect.Status &&
					!(expect.Status == prototype.StatusSuccess && actual.Status == prototype.StatusDeductStamina) {
					c.log.Errorf("InvoiceMismatch: expect_status=%d, status=%d, err=%s. trx #%d of block %d",
						expect.Status, actual.Status, actual.ErrorInfo, trxIdx, blockNum)
					invoiceOK = false
				}
				if actual.NetUsage != expect.NetUsage {
					c.log.Errorf("InvoiceMismatch: expect_net_usage=%d, net_usage=%d, trx #%d of block %d",
						expect.NetUsage, actual.NetUsage, trxIdx, blockNum)
					invoiceOK = false
				}
				if actual.CpuUsage != expect.CpuUsage {
					c.log.Errorf("InvoiceMismatch: expect_cpu_usage=%d, cpu_usage=%d, trx #%d of block %d",
						expect.CpuUsage, actual.CpuUsage, trxIdx, blockNum)
					invoiceOK = false
				}
			}
			if !invoiceOK {
				blockData, _ := json.MarshalIndent(blk, "", "  ")
				c.log.Errorf("InvalidBlock: block %d, marshal=%s", blockNum, string(blockData))
				mustSuccess(false, "mismatched invoice")
			}
		}
		pushTiming.SetPartial(time.Duration(applyTime))
		pushTiming.Mark()

		c.updateGlobalProperties(blk)

		pushTiming.Mark()

		ret, bpNameList := c.shuffle(blk)
		if ret {
			c.updateGlobalWitnessBoot(bpNameList)
			c.updateGlobalResourceParam(bpNameList)
			c.deleteUnusedBp(bpNameList)
		}

		pushTiming.Mark()

		c.updateAvgTps(blk)

		pushTiming.End()
		c.log.Debugf("PUSHBLOCK %d: %s, #tx=%d", blockNum, pushTiming.String(), totalCount)
	}

	afterTiming := common.NewTiming()
	eTiming := common.NewTiming()

	pseudoTrxObserver := c.stateObserver.NewTrxObserver()
	pseudoTrxObserver.BeginTrx("")

	afterTiming.Begin()

	c.createBlockSummary(blk)

	afterTiming.Mark()

	c.tm.BlockApplied(blk)

	afterTiming.Mark()

	eTiming.Begin()
	c.economist.Mint(pseudoTrxObserver)
	eTiming.Mark()
	c.economist.Do(pseudoTrxObserver)
	eTiming.Mark()
	c.economist.PowerDown()
	eTiming.End()
	c.log.Debugf("Economist: %s", eTiming.String())
	pseudoTrxObserver.EndTrx(true)

	afterTiming.Mark()

	c.notifyBlockApply(blk)

	afterTiming.End()
	c.log.Debugf("AFTER_BLOCK %d: %s", blockNum, afterTiming.String())
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
	pubKey, _ := prototype.PublicKeyFromWIF(constants.InitminerPubKey)
	name := &prototype.AccountName{Value: constants.COSInitMiner}
	newAccountWrap := table.NewSoAccountWrap(c.db, name)
	mustNoError(newAccountWrap.Create(func(tInfo *table.SoAccount) {
		tInfo.Name = name
		tInfo.CreatedTime = &prototype.TimePointSec{UtcSeconds: 0}
		tInfo.Balance = prototype.NewCoin(constants.COSInitSupply)
		tInfo.VestingShares = prototype.NewVest(0)
		tInfo.LastPostTime = &prototype.TimePointSec{UtcSeconds: 0}
		tInfo.LastVoteTime = &prototype.TimePointSec{UtcSeconds: 0}
		tInfo.NextPowerdownBlockNum = math.MaxUint32
		tInfo.EachPowerdownRate = &prototype.Vest{Value: 0}
		tInfo.ToPowerdown = &prototype.Vest{Value: 0}
		tInfo.HasPowerdown = &prototype.Vest{Value: 0}
		tInfo.Owner = pubKey
		tInfo.StakeVesting = prototype.NewVest(0)
	}), "CreateAccount error")

	// create witness_object
	witnessWrap := table.NewSoWitnessWrap(c.db, name)
	mustNoError(witnessWrap.Create(func(tInfo *table.SoWitness) {
		tInfo.Owner = name
		tInfo.CreatedTime = &prototype.TimePointSec{UtcSeconds: 0}
		tInfo.SigningKey = pubKey
		tInfo.Active = true
		tInfo.ProposedStaminaFree = constants.DefaultStaminaFree
		tInfo.TpsExpected = constants.DefaultTPSExpected
		tInfo.AccountCreateFee = prototype.NewCoin(constants.DefaultAccountCreateFee)
		tInfo.VoteCount = prototype.NewVest(0)
	}), "Witness Create Error")

	// create dynamic global properties
	dgpWrap := table.NewSoGlobalWrap(c.db, &SingleId)
	mustNoError(dgpWrap.Create(func(tInfo *table.SoGlobal) {
		tInfo.Id = SingleId
		tInfo.Props = &prototype.DynamicProperties{}
		tInfo.Props.CurrentWitness = name
		tInfo.Props.Time = &prototype.TimePointSec{UtcSeconds: constants.GenesisTime}
		tInfo.Props.HeadBlockId = &prototype.Sha256{Hash: make([]byte, 32)}
		// @ recent_slots_filled
		// @ participation_count
		tInfo.Props.CurrentSupply = prototype.NewCoin(constants.COSInitSupply)
		tInfo.Props.TotalCos = prototype.NewCoin(constants.COSInitSupply)
		tInfo.Props.MaximumBlockSize = constants.MaxBlockSize
		tInfo.Props.TotalUserCnt = 1
		tInfo.Props.StaminaFree = constants.DefaultStaminaFree
		tInfo.Props.TpsExpected = constants.DefaultTPSExpected
		tInfo.Props.AccountCreateFee = prototype.NewCoin(constants.DefaultAccountCreateFee)
		tInfo.Props.TotalVestingShares = prototype.NewVest(0)
		tInfo.Props.PostRewards = prototype.NewVest(0)
		tInfo.Props.ReplyRewards = prototype.NewVest(0)
		tInfo.Props.PostWeightedVps = "0"
		tInfo.Props.ReplyWeightedVps = "0"
		tInfo.Props.ReportRewards = prototype.NewVest(0)
		tInfo.Props.IthYear = 1
		tInfo.Props.AnnualBudget = prototype.NewVest(0)
		tInfo.Props.AnnualMinted = prototype.NewVest(0)
		tInfo.Props.PostDappRewards = prototype.NewVest(0)
		tInfo.Props.ReplyDappRewards = prototype.NewVest(0)
		tInfo.Props.VoterRewards = prototype.NewVest(0)
		tInfo.Props.StakeVestingShares = prototype.NewVest(0)
		tInfo.Props.OneDayStamina = constants.OneDayStamina
	}), "CreateDynamicGlobalProperties error")

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
		tInfo.CurrentShuffledWitness = append(tInfo.CurrentShuffledWitness, constants.COSInitMiner)
	}), "CreateWitnessScheduleObject error")
}

func (c *TrxPool) TransferToVest(value *prototype.Coin) {
	c.modifyGlobalDynamicData(func(dgpo *prototype.DynamicProperties) {
		cos := dgpo.GetTotalCos()
		vest := dgpo.GetTotalVestingShares()
		addVest := value.ToVest()

		mustNoError(cos.Sub(value), "TotalCos overflow")
		dgpo.TotalCos = cos

		mustNoError(vest.Add(addVest), "TotalVestingShares overflow")
		dgpo.TotalVestingShares = vest
	})
}

func (c *TrxPool) TransferFromVest(value *prototype.Vest) {
	c.modifyGlobalDynamicData(func(dgpo *prototype.DynamicProperties) {
		cos := dgpo.GetTotalCos()
		vest := dgpo.GetTotalVestingShares()
		addCos := value.ToCoin()

		mustNoError(cos.Add(addCos), "TotalCos overflow")
		dgpo.TotalCos = cos

		mustNoError(vest.Sub(value), "TotalVestingShares overflow")
		dgpo.TotalVestingShares = vest
	})
}

func (c *TrxPool) TransferToStakeVest(value *prototype.Coin) {
	c.modifyGlobalDynamicData(func(dgpo *prototype.DynamicProperties) {
		vest := dgpo.GetStakeVestingShares()
		addVest := value.ToVest()

		mustNoError(vest.Add(addVest), "StakeVestingShares overflow")
		dgpo.StakeVestingShares = vest
	})
}

func (c *TrxPool) TransferFromStakeVest(value *prototype.Vest) {
	c.modifyGlobalDynamicData(func(dgpo *prototype.DynamicProperties) {
		vest := dgpo.GetStakeVestingShares()

		mustNoError(vest.Sub(value), "UnStakeVestingShares overflow")
		dgpo.StakeVestingShares = vest
	})
}

func (c *TrxPool) validateBlockHeader(blk *prototype.SignedBlock) {
	headID := c.headBlockID()
	if !bytes.Equal(headID.Hash, blk.SignedHeader.Header.Previous.Hash) {
		c.log.Error("[trxpool]:", "validateBlockHeader Error: ", headID.ToString(), " prev:", blk.SignedHeader.Header.Previous.ToString())
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

	if c.enableBAH {
		ver, hash := c.iceberg.LatestBlockApplyHashUnpacked()
		bVer, bHash := common.UnpackBlockApplyHash(blk.SignedHeader.Header.PrevApplyHash)
		if ver != bVer {
			c.log.Warnf("BlockApplyHashWarn: version mismatch. block %d (by %s): %08x, me: %08x",
				blk.SignedHeader.Number(), blk.SignedHeader.Header.Witness.Value, bVer, ver)
		} else if hash != bHash {
			c.log.Errorf("BlockApplyHashError: block %d (by %s): %08x, me: %08x",
				blk.SignedHeader.Number(), blk.SignedHeader.Header.Witness.Value, bHash, hash)
			panic("block apply hash not equal")
		}
	}
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

func (c *TrxPool) updateGlobalDataToDB(dgpo *prototype.DynamicProperties) {
	dgpWrap := table.NewSoGlobalWrap(c.db, &SingleId)
	mustSuccess(dgpWrap.MdProps(dgpo), "")
}

func (c *TrxPool) modifyGlobalDynamicData(f func(props *prototype.DynamicProperties)) {
	dgpWrap := table.NewSoGlobalWrap(c.db, &SingleId)
	props := dgpWrap.GetProps()

	f(props)

	mustSuccess(dgpWrap.MdProps(props), "")
}

func (c *TrxPool) ModifyProps(modifier func(oldProps *prototype.DynamicProperties)) {
	c.modifyGlobalDynamicData(modifier)
}

func (c *TrxPool) updateAvgTps(blk *prototype.SignedBlock) {
	dgpWrap := table.NewSoGlobalWrap(c.db, &constants.GlobalId)
	props := dgpWrap.GetProps()
	tpsInWindow := props.GetAvgTpsInWindow()
	lastUpdate := props.GetAvgTpsUpdateBlock()
	oneDayStamina := props.GetOneDayStamina()
	expectedTps := props.GetTpsExpected()

	newOneDayStamina,tpsInWindowNew := c.resourceLimiter.UpdateDynamicStamina(tpsInWindow,oneDayStamina, uint64(len(blk.Transactions)),lastUpdate,blk.Id().BlockNum(),expectedTps)
	c.ModifyProps(func(props *prototype.DynamicProperties) {
		props.OneDayStamina = newOneDayStamina
		props.AvgTpsInWindow = tpsInWindowNew
		props.AvgTpsUpdateBlock = blk.Id().BlockNum()
	})
}

func (c *TrxPool) updateGlobalWitnessBoot(bpNameList []string) {
	if len(bpNameList) == 1 && bpNameList[0] == constants.COSInitMiner {
		return
	}
	c.modifyGlobalDynamicData(func(dgpo *prototype.DynamicProperties) {
		dgpo.WitnessBootCompleted = true
	})
}

func (c *TrxPool) updateGlobalResourceParam(bpNameList []string) {
	var tpsExpectedList  []uint64
	var staminaFreeList  []uint64
	var accountCreationFee []uint64

	for i := range bpNameList {
		ac := &prototype.AccountName{
			Value: bpNameList[i],
		}
		witnessWrap := table.NewSoWitnessWrap(c.db, ac)
		if !witnessWrap.CheckExist() {
			c.log.Fatalf("witness %v doesn't exist", bpNameList[i])
		}
		tpsExpectedList = append(tpsExpectedList, witnessWrap.GetTpsExpected())
		staminaFreeList = append(staminaFreeList, witnessWrap.GetProposedStaminaFree())
		accountCreationFee = append(accountCreationFee, witnessWrap.GetAccountCreateFee().Value)
	}

	sort.Sort(selfmath.DirRange(tpsExpectedList))
	sort.Sort(selfmath.DirRange(staminaFreeList))
	sort.Sort(selfmath.DirRange(accountCreationFee))

	c.modifyGlobalDynamicData(func(dgpo *prototype.DynamicProperties) {
		dgpo.StaminaFree = staminaFreeList[ len(staminaFreeList) / 2 ]
		dgpo.TpsExpected = tpsExpectedList[ len(tpsExpectedList) / 2 ]
		midVal := accountCreationFee[ len(accountCreationFee) / 2 ]
		dgpo.AccountCreateFee = prototype.NewCoin(midVal)
	})
}

func (c *TrxPool) deleteUnusedBp(bpNameList []string) {
	// delete unActive bp
	revList := table.SWitnessVoteCountWrap{Dba: c.db}
	var deletelist       []*prototype.AccountName

	_ = revList.ForEachByRevOrder(nil, nil,nil,nil, func(mVal *prototype.AccountName, sVal *prototype.Vest, idx uint32) bool {
		if mVal != nil {
			witnessWrap := table.NewSoWitnessWrap(c.db, mVal)
			if witnessWrap.CheckExist() {
				if !witnessWrap.GetActive() {
					deletelist = append(deletelist, mVal)
				}
			}
		}
		return true
	})

	for i:=0;i<len(deletelist);i++ {
		witnessWrap := table.NewSoWitnessWrap(c.db, deletelist[i])
		mustSuccess(witnessWrap.RemoveWitness(), fmt.Sprintf("delete unregister bp %s error", deletelist[i].Value))
	}

	// maybe delete bp constants.COSInitMiner
	if len(bpNameList) > 1 || (len(bpNameList) == 1 && bpNameList[0] != constants.COSInitMiner) {
		ac := &prototype.AccountName{
			Value: constants.COSInitMiner,
		}
		witnessWrap := table.NewSoWitnessWrap(c.db, ac)
		if witnessWrap.CheckExist() {
			payBackVoteCntToVoter(c.db, ac)
			mustSuccess(witnessWrap.RemoveWitness(), fmt.Sprintf("delete bp %s error", constants.COSInitMiner))
		}
	}
}

func (c *TrxPool) updateGlobalProperties(blk *prototype.SignedBlock) {
	id := blk.Id()
	blockID := &prototype.Sha256{Hash: id.Data[:]}

	c.modifyGlobalDynamicData(func(dgpo *prototype.DynamicProperties) {
		dgpo.HeadBlockNumber = blk.Id().BlockNum()
		dgpo.HeadBlockId = blockID
		dgpo.HeadBlockPrefix = binary.BigEndian.Uint32(id.Data[8:12])
		dgpo.Time = blk.SignedHeader.Header.Timestamp
		dgpo.CurrentWitness = blk.SignedHeader.Header.Witness

		trxCount := len(blk.Transactions)
		dgpo.TotalTrxCnt += uint64(trxCount)
		dgpo.Tps = uint32(trxCount / constants.BlockInterval)

		if dgpo.MaxTps < dgpo.Tps {
			dgpo.MaxTps = dgpo.Tps
			dgpo.MaxTpsBlockNum = blk.Id().BlockNum()
		}

		c.log.Debugf("UPDATEDGP %d: headNum=%d, headId=%v", id.BlockNum(), dgpo.HeadBlockNumber, dgpo.HeadBlockId.Hash)
	})

	c.noticer.Publish(constants.NoticeAddTrx, blk)
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

func (c *TrxPool) GetSigningPubKey(witness string) *prototype.PublicKeyType {
	ac := &prototype.AccountName{
		Value: witness,
	}
	witnessWrap := table.NewSoWitnessWrap(c.db, ac)
	if !witnessWrap.CheckExist() {
		return nil
	}
	return witnessWrap.GetSigningKey()
}

func (c *TrxPool) GetWitnessTopN(n uint32) ([]string, []*prototype.PublicKeyType) {
	var names            []string
	var bpNames          []string
	var keys             []*prototype.PublicKeyType
	revList := table.SWitnessVoteCountWrap{Dba: c.db}
	var bpCount uint32 = 0
	_ = revList.ForEachByRevOrder(nil, nil,nil,nil, func(mVal *prototype.AccountName, sVal *prototype.Vest, idx uint32) bool {
		if mVal != nil {
			witnessWrap := table.NewSoWitnessWrap(c.db, mVal)
			if witnessWrap.CheckExist() {
				if witnessWrap.GetActive() {
					bpCount++
					names = append(names, mVal.Value)
				}
			}
		}
		if bpCount < n + 1 {
			return true
		}
		//if idx < n {
		//	return true
		//}
		return false
	})

	for i := range names {
		if names[i] == constants.COSInitMiner && len(names) > 1 {
			continue
		}
		ac := &prototype.AccountName{
			Value: names[i],
		}
		witnessWrap := table.NewSoWitnessWrap(c.db, ac)
		if !witnessWrap.CheckExist() {
			c.log.Fatalf("witness %v doesn't exist", names[i])
		}
		dbPubKey := witnessWrap.GetSigningKey()
		keys = append(keys, dbPubKey)
		bpNames = append(bpNames, names[i])

		if uint32(len(bpNames)) == n {
			break
		}
	}


	//return names, keys
	return bpNames, keys
}

func (c *TrxPool) SetShuffledWitness(names []string, keys []*prototype.PublicKeyType) {
	witnessScheduleWrap := table.NewSoWitnessScheduleObjectWrap(c.db, &SingleId)
	mustSuccess(witnessScheduleWrap.MdCurrentShuffledWitness(names), "SetWitness error")
	mustSuccess(witnessScheduleWrap.MdPubKey(keys), "set witness pub key failed")
}

func (c *TrxPool) GetShuffledWitness() ([]string, []*prototype.PublicKeyType) {
	witnessScheduleWrap := table.NewSoWitnessScheduleObjectWrap(c.db, &SingleId)
	return witnessScheduleWrap.GetCurrentShuffledWitness(), witnessScheduleWrap.GetPubKey()
}

func (c *TrxPool) PopBlock(num uint64) error {
	c.db.Lock()
	defer c.db.Unlock()

	err := c.iceberg.RevertBlock(num)
	if err != nil {
		c.log.Errorf("PopBlock %d failed, error: %v", num, err)
	}

	c.tm.BlockReverted(num)

	return err
}

func (c *TrxPool) Commit(num uint64) {
	s := time.Now()
	c.db.Lock()
	defer func() {
		c.db.Unlock()
		c.log.Debug("[trxpool] Commit cost: ", time.Now().Sub(s))
	}()
	// this block can not be revert over, so it's irreversible
	err := c.iceberg.FinalizeBlock(num)
	mustSuccess(err == nil, fmt.Sprintf("commit block: %d, error is %v", num, err))

	c.tm.BlockCommitted(num)
}

func (c *TrxPool) GetLastPushedBlockNum() (uint64, error) {
	num, _, err := c.iceberg.LatestBlock()
	return num, err
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
	cmtNum, err := c.GetLastPushedBlockNum()
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
		cmtNum, err := c.GetLastPushedBlockNum()
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

func (c *TrxPool) calculateUserMaxStamina(db iservices.IDatabaseRW,name string) uint64 {
	dgpWrap := table.NewSoGlobalWrap(db, &SingleId)
	accountWrap := table.NewSoAccountWrap(db, &prototype.AccountName{Value:name})

	oneDayStamina := dgpWrap.GetProps().GetOneDayStamina()
	stakeVest := accountWrap.GetStakeVesting().Value

	allStakeVest := dgpWrap.GetProps().StakeVestingShares.Value
	if allStakeVest == 0 {
		return 0
	}

	stakeBig := big.NewInt(int64(stakeVest))
	allStakeVestBig := big.NewInt(int64(allStakeVest))
	oneDayStaminaBig := big.NewInt(int64(oneDayStamina))
	precision := big.NewInt(constants.LimitPrecision)

	stakeBig.Mul(stakeBig,precision)
	stakeBig.Mul(stakeBig,oneDayStaminaBig)
	stakeBig.Div(stakeBig,allStakeVestBig)
	return stakeBig.Div(stakeBig,precision).Uint64()
}

func (c *TrxPool) CalculateUserMaxStamina(db iservices.IDatabaseRW,name string) uint64 {
	return c.calculateUserMaxStamina(db,name)
}

func (c *TrxPool) CheckNetForRPC(name string, db iservices.IDatabaseRW, sizeInBytes uint64) (bool,uint64,uint64) {
	netUse := sizeInBytes * constants.NetConsumePointNum
	accountWrap := table.NewSoAccountWrap(db, &prototype.AccountName{Value:name})
	if !accountWrap.CheckExist() {
		return false,0,0
	}
	maxStamina := c.calculateUserMaxStamina(db,name)
	dgpWraper := table.NewSoGlobalWrap(db, &constants.GlobalId)
	freeOver,freeLeft := c.resourceLimiter.GetFreeLeft(dgpWraper.GetProps().GetStaminaFree(),accountWrap.GetStaminaFree(), accountWrap.GetStaminaFreeUseBlock(), c.GetProps().HeadBlockNumber)
	stakeOver,stakeLeft := c.resourceLimiter.GetStakeLeft(accountWrap.GetStamina(), accountWrap.GetStaminaUseBlock(), c.GetProps().HeadBlockNumber,maxStamina)
	if freeLeft >= netUse {
		return true,freeLeft+stakeLeft,netUse
	} else {
		if stakeLeft >= netUse-freeLeft {
			return true,freeLeft+stakeLeft,netUse
		} else {
			if freeOver == constants.FreeStaminaOverFlow || stakeOver == constants.StakeStaminaOverFlow {
				// overflow happened, let it go, we will update user's stamina
				return true, freeLeft+stakeLeft, netUse
			}
			return false, freeLeft+stakeLeft, netUse
		}
	}
}
