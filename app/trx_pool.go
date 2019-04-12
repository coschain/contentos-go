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
	"github.com/coschain/contentos-go/economist"
	"math"

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
	skip    prototype.SkipFlag
	shuffle                common.ShuffleFunc

	iceberg   *BlockIceberg
	economist *economist.Economist
	tm *TrxMgr
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
	c.economist = economist.New(c.db, c.noticer, &SingleId, c.log)
	dgpWrap := table.NewSoGlobalWrap(c.db, &SingleId)
	if !dgpWrap.CheckExist() {

		mustNoError(c.db.DeleteAll(), "truncate database error")

		//c.log.Info("start initGenesis")
		c.initGenesis()

		mustNoError(c.db.TagRevision(c.db.GetRevision(), GENESIS_TAG), "genesis tagging failed")
		c.iceberg = NewBlockIceberg(c.db)
		c.economist = economist.New(c.db, c.noticer, &SingleId, c.log)
		//c.log.Info("finish initGenesis")
	}
	commit, _ := c.iceberg.LastFinalizedBlock()
	latest, _, _ := c.iceberg.LatestBlock()
	c.tm = NewTrxMgr(c.db, c.log, latest, commit)
}

func (c *TrxPool) Stop() error {
	return nil
}

func (c *TrxPool) PushTrxToPending(trx *prototype.SignedTransaction) (err error) {
	return c.tm.AddTrx(trx, nil)
}

func (c *TrxPool) PushTrx(trx *prototype.SignedTransaction) (invoice *prototype.TransactionReceiptWithInfo) {
	rc := make(chan *prototype.TransactionReceiptWithInfo)
	_ = c.tm.AddTrx(trx, func(result *prototype.EstimateTrxResult) {
		rc <- result.Receipt
	})
	return <-rc
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
	//var err error = nil
	oldFlag := c.skip
	c.skip = skip

	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {
			case error:
				err = x
				//c.log.Errorf("push block error : %v", x.Error())
			case string:
				err = errors.New(x)
				//c.log.Errorf("push block error : %v ", x)
			default:
				err = errors.New("unknown panic type")
			}
			// undo changes
			_ = c.iceberg.EndBlock(false)
			c.log.Debug("ICEBERG: EndBlock FALSE")
			c.log.Errorf("push block fail,the error is %v,the block num is %v", r, blk.Id().BlockNum())
			//fmt.Printf("push block fail,the error is %v,the block num is %v \n", r, blk.Id().BlockNum())
		}
		c.skip = oldFlag

	}()

	if skip&prototype.Skip_apply_transaction == 0 {
		blkNum := blk.Id().BlockNum()
		c.log.Debugf("ICEBERG: BeginBlock %d", blkNum)
		_ = c.iceberg.BeginBlock(blkNum)
		c.applyBlock(blk, skip)
		c.log.Debug("ICEBERG: EndBlock TRUE")
		_ = c.iceberg.EndBlock(true)
	} else {
		// we have do a BeginTransaction at GenerateBlock
		c.applyBlock(blk, skip)
		c.log.Debug("ICEBERG: EndBlock TRUE")
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
			if err = c.pushBlockNoLock(newBlock, c.skip|prototype.Skip_apply_transaction|prototype.Skip_block_check); err != nil {
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
	oldSkip := c.skip

	t0 := time.Now()

	defer func() {
		c.skip = oldSkip
		if err := recover(); err != nil {
			c.log.Debug("ICEBERG: EndBlock FALSE")
			_ = c.iceberg.EndBlock(false)

			//c.log.Errorf("GenerateBlock Error: %v", err)
			//panic(err)
			b, e = nil, fmt.Errorf("%v", err)
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

	dgpWrap := table.NewSoGlobalWrap(c.db, &SingleId)
	maxBlockSize := dgpWrap.GetProps().MaximumBlockSize

	signBlock := &prototype.SignedBlock{}
	signBlock.SignedHeader = &prototype.SignedBlockHeader{}
	signBlock.SignedHeader.Header = &prototype.BlockHeader{}
	//c.currentTrxInBlock = 0

	blkNum := c.headBlockNum() + 1
	c.log.Debugf("ICEBERG: BeginBlock %d", blkNum)
	_ = c.iceberg.BeginBlock(blkNum)

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
	t1 := time.Now()
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
				signBlock.Transactions = append(signBlock.Transactions, result.ToTrxWrapper())
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
	t2 := time.Now()

	signBlock.SignedHeader.Header.Previous = pre
	signBlock.SignedHeader.Header.Timestamp = &prototype.TimePointSec{UtcSeconds: timestamp}
	id := signBlock.CalculateMerkleRoot()
	signBlock.SignedHeader.Header.TransactionMerkleRoot = &prototype.Sha256{Hash: id.Data[:]}
	signBlock.SignedHeader.Header.Witness = &prototype.AccountName{Value: witness}
	signBlock.SignedHeader.WitnessSignature = &prototype.SignatureType{}
	_ = signBlock.SignedHeader.Sign(priKey)

	mustSuccess(proto.Size(signBlock) <= constants.MaxBlockSize, "block size too big")

	if len(failedTrx) > 0 {
		c.tm.ReturnTrx(failedTrx...)
	}

	t3 := time.Now()
	c.updateGlobalProperties(signBlock)
	t4 := time.Now()
	c.shuffle(signBlock)
	t5 := time.Now()

	c.log.Debugf("GENBLOCK %d: %v|%v|%v(%v)|%v|%v|%v, timeout=%v, pending=%d, failed=%d, inblk=%d",
		signBlock.Id().BlockNum(),
		t5.Sub(t0), t1.Sub(t0), t2.Sub(t1), time.Duration(applyTime), t3.Sub(t2), t4.Sub(t3), t5.Sub(t4), timeOut,
		c.tm.WaitingCount(), len(failedTrx), len(signBlock.Transactions))

	b, e = signBlock, nil
	return
}

func (c *TrxPool) notifyBlockApply(block *prototype.SignedBlock) {
	t0 := time.Now()
	for _, trx := range block.Transactions {
		for _, op := range trx.SigTrx.Trx.Operations {
			c.noticer.Publish(constants.NoticeOpPost, &prototype.OperationNotification{Op: op})
		}
		c.noticer.Publish(constants.NoticeTrxPost, trx.SigTrx)
	}
	t1 := time.Now()
	c.noticer.Publish(constants.NoticeBlockApplied, block)
	t2 := time.Now()
	c.log.Debugf("NOTIFYBLOCK %d: %v|%v|%v, #tx=%d", block.Id().BlockNum(), t2.Sub(t0), t1.Sub(t0), t2.Sub(t1), len(block.Transactions))
}

func (c *TrxPool) notifyTrxApplyResult(trx *prototype.SignedTransaction, res bool,
	receipt *prototype.TransactionReceiptWithInfo) {
	c.noticer.Publish(constants.NoticeTrxApplied, trx, receipt)
}

func (c *TrxPool) applyTransactionOnDb(db iservices.IDatabaseRW, entry *TrxEntry) {
	result := entry.GetTrxResult()
	receipt, sigTrx := result.GetReceipt(), result.GetSigTrx()

	defer func() {
		if err := recover(); err != nil {
			receipt.Status = prototype.StatusError
			receipt.ErrorInfo = fmt.Sprintf("applyTransaction failed : %v", err)
			c.notifyTrxApplyResult(sigTrx, false, receipt)
			panic(receipt.ErrorInfo)
		} else {
			receipt.Status = prototype.StatusSuccess
			c.notifyTrxApplyResult(sigTrx, true, receipt)
			return
		}
	}()

	trxContext := NewTrxContextWithSigningKey(result, db, entry.GetTrxSigningKey())
	for _, op := range sigTrx.Trx.Operations {
		trxContext.StartNextOp()
		c.applyOperation(trxContext, op)
	}
}

func (c *TrxPool) applyOperation(trxCtx *TrxContext, op *prototype.Operation) {
	// @ not use yet
	//n := &prototype.OperationNotification{Op: op}
	//c.notifyOpPreExecute(n)

	eva := c.getEvaluator(trxCtx, op)
	eva.Apply()

	// @ not use yet
	//c.notifyOpPostExecute(n)
}

func (c *TrxPool) getEvaluator(trxCtx *TrxContext, op *prototype.Operation) BaseEvaluator {
	ctx := &ApplyContext{db: trxCtx.db, control: trxCtx, vmInjector: trxCtx}
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

	if skip & prototype.Skip_block_check == 0 {
		merkleRoot := blk.CalculateMerkleRoot()
		mustSuccess(bytes.Equal(merkleRoot.Data[:], blk.SignedHeader.Header.TransactionMerkleRoot.Hash), "Merkle check failed")

		// validate_block_header
		c.validateBlockHeader(blk)

		//c.currentBlockNum = nextBlockNum
		//c.currentTrxInBlock = 0

		blockSize := proto.Size(blk)
		mustSuccess(uint32(blockSize) <= c.GetProps().GetMaximumBlockSize(), "Block size is too big")

		if uint32(blockSize) < constants.MinBlockSize {
			// elog("Block size is too small")
		}
	}

	// @ process extension

	// @ hardfork_state

	if skip&prototype.Skip_apply_transaction == 0 {
		t0 := time.Now()

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
			for j := 0; j < d; j++ {
				mustSuccess(entries[i + j].GetTrxResult().Receipt.Status == blk.Transactions[i + j].Invoice.Status, "mismatched invoice")
			}
		}
		t1 := time.Now()
		c.updateGlobalProperties(blk)
		t2 := time.Now()
		c.shuffle(blk)
		t3 := time.Now()
		c.log.Debugf("PUSHBLOCK %d: %v|%v(%v)|%v|%v, #tx=%d", blk.Id().BlockNum(),
			t3.Sub(t0), t1.Sub(t0), time.Duration(applyTime), t2.Sub(t1), t3.Sub(t2), totalCount)
	}
	t0 := time.Now()
	c.createBlockSummary(blk)
	t1 := time.Now()
	c.tm.BlockApplied(blk)
	t2 := time.Now()

	tinit := time.Now()
	c.economist.Mint()
	tmint := time.Now()
	c.economist.Do()
	tdo := time.Now()
	c.economist.PowerDown()
	tpd := time.Now()
	c.log.Debugf("Economist: %v|%v|%v|%v", tpd.Sub(tinit), tmint.Sub(tinit), tdo.Sub(tmint), tpd.Sub(tdo))

	t3 := time.Now()
	c.notifyBlockApply(blk)
	t4 := time.Now()

	c.log.Debugf("AFTER_BLOCK %d: %v|%v|%v|%v|%v", blk.Id().BlockNum(),
		t4.Sub(t0), t1.Sub(t0), t2.Sub(t1), t3.Sub(t2), t4.Sub(t3))
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
		tInfo.Balance = prototype.NewCoin(constants.COSInitSupply - 1000)
		tInfo.VestingShares = prototype.NewVest(1000)
		tInfo.LastPostTime = &prototype.TimePointSec{UtcSeconds: 0}
		tInfo.LastVoteTime = &prototype.TimePointSec{UtcSeconds: 0}
		tInfo.NextPowerdownBlockNum = math.MaxUint32
		tInfo.EachPowerdownRate = &prototype.Vest{Value: 0}
		tInfo.ToPowerdown = &prototype.Vest{Value: 0}
		tInfo.HasPowerdown = &prototype.Vest{Value: 0}
		tInfo.Owner = pubKey
	}), "CreateAccount error")

	// create account authority
	//authorityWrap := table.NewSoAccountAuthorityObjectWrap(c.db, name)
	//ownerAuth := prototype.NewAuthorityFromPubKey(pubKey)

	//mustNoError(authorityWrap.Create(func(tInfo *table.SoAccountAuthorityObject) {
	//	tInfo.Account = name
	//	tInfo.Owner = ownerAuth
	//}), "CreateAccountAuthorityObject error ")

	// create witness_object
	witnessWrap := table.NewSoWitnessWrap(c.db, name)
	mustNoError(witnessWrap.Create(func(tInfo *table.SoWitness) {
		tInfo.Owner = name
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
		tInfo.Props.Time = &prototype.TimePointSec{UtcSeconds: constants.GenesisTime}
		tInfo.Props.HeadBlockId = &prototype.Sha256{Hash: make([]byte, 32)}
		// @ recent_slots_filled
		// @ participation_count
		tInfo.Props.CurrentSupply = prototype.NewCoin(constants.COSInitSupply)
		tInfo.Props.TotalCos = prototype.NewCoin(constants.COSInitSupply)
		tInfo.Props.MaximumBlockSize = constants.MaxBlockSize
		tInfo.Props.TotalUserCnt = 1
		tInfo.Props.TotalVestingShares = prototype.NewVest(0)
		tInfo.Props.PostRewards = prototype.NewVest(0)
		tInfo.Props.ReplyRewards = prototype.NewVest(0)
		tInfo.Props.PostWeightedVps = 0
		tInfo.Props.ReplyWeightedVps = 0
		tInfo.Props.ReportRewards = prototype.NewVest(0)
		tInfo.Props.IthYear = 1
		tInfo.Props.AnnualBudget = prototype.NewVest(0)
		tInfo.Props.AnnualMinted = prototype.NewVest(0)
		tInfo.Props.PostDappRewards = prototype.NewVest(0)
		tInfo.Props.ReplyDappRewards = prototype.NewVest(0)
		tInfo.Props.VoterRewards = prototype.NewVest(0)
	}), "CreateDynamicGlobalProperties error")

	//create rewards keeper
	//keeperWrap := table.NewSoRewardsKeeperWrap(c.db, &SingleId)
	//rewards := make(map[string]*prototype.Vest)
	//rewards["initminer"] = &prototype.Vest{Value: 0}
	//mustNoError(keeperWrap.Create(func(tInfo *table.SoRewardsKeeper) {
	//	tInfo.Id = SingleId
	//	tInfo.Keeper.Rewards = map[string]*prototype.Vest{}
		//tInfo.Keeper = &prototype.InternalRewardsKeeper{Rewards: rewards}
	//}), "Create Rewards Keeper error")

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
	// this check is useful ?
	//mustSuccess(dgpo.GetHeadBlockNumber()-dgpo.GetIrreversibleBlockNum() < constants.MaxUndoHistory, "The database does not have enough undo history to support a blockchain with so many missed blocks.")
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

func (c *TrxPool) GetWitnessTopN(n uint32) []string {
	var ret []string
	revList := table.SWitnessVoteCountWrap{Dba: c.db}
	_ = revList.ForEachByRevOrder(nil, nil,nil,nil, func(mVal *prototype.AccountName, sVal *uint64, idx uint32) bool {
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
