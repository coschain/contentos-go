package core

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/app"
	"github.com/coschain/contentos-go/app/plugins"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/common/eventloop"
	"github.com/coschain/contentos-go/config"
	"github.com/coschain/contentos-go/db/storage"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/prototype"
	"github.com/gogo/protobuf/proto"
	"github.com/hashicorp/golang-lru"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

const sTrxReceiptCacheSize = 10000

type DandelionCore struct {
	node *node.Node
	cfg node.Config
	chainId prototype.ChainId
	timeStamp uint32
	prevHash *prototype.Sha256
	accounts map[string]*prototype.PrivateKeyType
	trxReceipts *lru.Cache
	beforePreshuffle, afterPreshuffle map[string]func()
}

func NewDandelionCore(logger *logrus.Logger, enablePlugins bool, sqlPlugins []string) *DandelionCore {
	if logger == nil {
		logger = logrus.New()
		logger.SetOutput(ioutil.Discard)
	}

	cfg := config.DefaultNodeConfig
	cfg.ChainId = "dandelion"
	cfg.Name = "dandelionNode"
	buf := make([]byte, 8)
	_, _ = rand.Reader.Read(buf)
	cfg.DataDir = filepath.Join(os.TempDir(), hex.EncodeToString(buf))

	n, _ := node.New(&cfg)
	n.Log = logger
	n.MainLoop = eventloop.NewEventLoop()
	n.EvBus = EventBus.New()

	pluginMgr := plugins.NewPluginMgt(sqlPlugins)

	_ = n.Register(iservices.DbServerName, func(ctx *node.ServiceContext) (node.Service, error) {
		return storage.NewGuardedDatabaseService(ctx, "./db/")
	})

	if enablePlugins {
		pluginMgr.RegisterSQLServices(n, &cfg)
	}

	_ = n.Register(iservices.TxPoolServerName, func(ctx *node.ServiceContext) (node.Service, error) {
		return app.NewController(ctx, n.Log)
	})

	if enablePlugins {
		pluginMgr.RegisterTrxPoolDependents(n, &cfg)
	}

	_ = n.Register(DummyConsensusName, func(ctx *node.ServiceContext) (node.Service, error) {
		return NewDummyConsensus(ctx)
	})

	receiptCache, _ := lru.New(sTrxReceiptCacheSize)
	d := &DandelionCore{
		node: n,
		cfg: cfg,
		chainId: prototype.ChainId{ Value: common.GetChainIdByName(cfg.ChainId) },
		timeStamp: uint32(time.Now().Unix()),
		prevHash: &prototype.Sha256{ Hash: make([]byte, 32) },
		accounts: make(map[string]*prototype.PrivateKeyType),
		trxReceipts: receiptCache,
		beforePreshuffle: make(map[string]func()),
		afterPreshuffle: make(map[string]func()),
	}

	initminerKey, _ := prototype.PrivateKeyFromWIF(constants.InitminerPrivKey)
	d.PutAccount(constants.COSInitMiner, initminerKey)

	return d
}

func (d *DandelionCore) cleanup() {
	_ = os.RemoveAll(d.cfg.DataDir)
}

func (d *DandelionCore) Start() (err error) {
	defer func() {
		if err != nil {
			d.cleanup()
		}
	}()
	_ = os.RemoveAll(d.cfg.DataDir)
	_ = os.Mkdir(d.cfg.DataDir, 0777)
	_ = os.Mkdir(filepath.Join(d.cfg.DataDir, d.cfg.Name), 0777)

	if err = d.node.Start(); err == nil {
		// produce the first block with no transactions.
		// this will set correct head timestamp in state db.
		err = d.ProduceBlocks(1)
		if err == nil {
			_ = d.node.EvBus.Subscribe(constants.NoticeTrxApplied, d.trxApplied)
			_ = d.node.EvBus.Subscribe(BeforePreShuffleEvent, d.beforePreShuffle)
			_ = d.node.EvBus.Subscribe(AfterPreShuffleEvent, d.afterPreShuffle)
		}
	}
	return
}

func (d *DandelionCore) Stop() error {
	defer d.cleanup()
	_ = d.node.EvBus.Unsubscribe(constants.NoticeTrxApplied, d.trxApplied)
	_ = d.node.EvBus.Unsubscribe(BeforePreShuffleEvent, d.beforePreShuffle)
	_ = d.node.EvBus.Unsubscribe(AfterPreShuffleEvent, d.afterPreShuffle)
	return d.node.Stop()
}

func (d *DandelionCore) Node() *node.Node {
	return d.node
}

func (d *DandelionCore) NodeConfig() *node.Config {
	return &d.cfg
}

func (d *DandelionCore) Database() iservices.IDatabaseService {
	if s, err := d.node.Service(iservices.DbServerName); err != nil {
		return nil
	} else {
		return s.(iservices.IDatabaseService)
	}
}

func (d *DandelionCore) TrxPool() iservices.ITrxPool {
	if s, err := d.node.Service(iservices.TxPoolServerName); err != nil {
		return nil
	} else {
		return s.(iservices.ITrxPool)
	}
}

func (d *DandelionCore) Consensus() *DummyConsensus {
	if s, err := d.node.Service(DummyConsensusName); err != nil {
		return nil
	} else {
		return s.(*DummyConsensus)
	}
}

func (d *DandelionCore) Head() (blockId common.BlockID) {
	copy(blockId.Data[:], d.prevHash.Hash)
	return
}

func (d *DandelionCore) PutAccount(name string, key *prototype.PrivateKeyType) {
	d.accounts[name] = key
}

func (d *DandelionCore) GetAccountKey(name string) *prototype.PrivateKeyType {
	return d.accounts[name]
}

func (d *DandelionCore) produceBlock() (block *prototype.SignedBlock, err error) {
	const skip = prototype.Skip_block_signatures
	var blockId common.BlockID

	copy(blockId.Data[:], d.prevHash.Hash)
	num := blockId.BlockNum() + 1
	bp := d.Consensus().GetProducer(num)
	bpKey, ok := d.accounts[bp]
	if !ok {
		err = fmt.Errorf("unknown block producer: %s", bp)
		return
	}
	if block, err = d.TrxPool().GenerateAndApplyBlock(bp, d.prevHash, d.timeStamp, bpKey, skip); err != nil {
		return
	}
	blockId = block.Id()
	d.TrxPool().Commit(num)
	copy(d.prevHash.Hash, blockId.Data[:])
	d.timeStamp += constants.BlockInterval
	d.node.EvBus.Publish(constants.NoticeLibChange, []common.ISignedBlock{block})
	return
}

func (d *DandelionCore) ProduceBlocks(count int) error {
	for i := 0; i < count; i++ {
		if _, err := d.produceBlock(); err != nil {
			return err
		}
	}
	return nil
}

func (d *DandelionCore) trxApplied(trx *prototype.SignedTransaction, result *prototype.TransactionReceiptWithInfo, blockNum uint64) {
	d.trxReceipts.Add(string(trx.Signature.Sig), result)
}

func (d *DandelionCore) GetTrxReceipt(trx *prototype.SignedTransaction) *prototype.TransactionReceiptWithInfo {
	if r, ok := d.trxReceipts.Get(string(trx.Signature.Sig)); ok {
		return r.(*prototype.TransactionReceiptWithInfo)
	}
	return nil
}


func (d *DandelionCore) SendRawTrx(signedTrx *prototype.SignedTransaction) (*prototype.SignedTransaction, error) {
	err := d.TrxPool().PushTrxToPending(signedTrx)
	return signedTrx, err
}

func (d *DandelionCore) sendTrx(privateKey *prototype.PrivateKeyType, operations...*prototype.Operation) (*prototype.SignedTransaction, error) {
	data, err := proto.Marshal(&prototype.Transaction{
		RefBlockNum: common.TaposRefBlockNum(d.Head().BlockNum()),
		RefBlockPrefix: common.TaposRefBlockPrefix(d.prevHash.Hash),
		Expiration: prototype.NewTimePointSec(d.timeStamp + constants.TrxMaxExpirationTime - 1),
		Operations: operations,
	},)
	if err != nil {
		return nil, err
	}
	trx := new(prototype.Transaction)
	if err = proto.Unmarshal(data, trx); err != nil {
		return nil, err
	}
	signedTrx := &prototype.SignedTransaction{
		Trx: trx,
		Signature: new(prototype.SignatureType),
	}
	signedTrx.Signature.Sig = signedTrx.Sign(privateKey, d.chainId)
	err = d.TrxPool().PushTrxToPending(signedTrx)
	return signedTrx, err
}

func (d *DandelionCore) sendTrxAndProduceBlock(privateKey *prototype.PrivateKeyType, operations...*prototype.Operation) (trx *prototype.SignedTransaction, block *prototype.SignedBlock, err error) {
	if trx, err = d.sendTrx(privateKey, operations...); err != nil {
		return
	}
	block, err = d.produceBlock()
	return
}

func (d *DandelionCore) SendTrx(privateKey *prototype.PrivateKeyType, operations...*prototype.Operation) error {
	_, err := d.sendTrx(privateKey, operations...)
	return err
}

func (d *DandelionCore) SendTrxByAccount(name string, operations...*prototype.Operation) error {
	key, ok := d.accounts[name]
	if !ok {
		return fmt.Errorf("unknown account: %s", name)
	}
	return d.SendTrx(key, operations...)
}

func (d *DandelionCore) SendTrxEx2(privateKey *prototype.PrivateKeyType, operations...*prototype.Operation) (*prototype.SignedTransaction, *prototype.TransactionReceiptWithInfo, error) {
	trx, block, err := d.sendTrxAndProduceBlock(privateKey, operations...)
	if err != nil {
		return nil, nil, err
	}
	for _, w := range block.Transactions {
		if bytes.Compare(w.SigTrx.Signature.Sig, trx.Signature.Sig) != 0 {
			continue
		}
		if r, ok := d.trxReceipts.Get(string(trx.Signature.Sig)); ok {
			return trx, r.(*prototype.TransactionReceiptWithInfo), nil
		}
	}
	return trx, nil, errors.New("transaction not found in block")
}

func (d *DandelionCore) SendTrxEx(privateKey *prototype.PrivateKeyType, operations...*prototype.Operation) (*prototype.TransactionReceiptWithInfo, error) {
	_, r, err := d.SendTrxEx2(privateKey, operations...)
	return r, err
}

func (d *DandelionCore) SendTrxByAccountEx(name string, operations...*prototype.Operation) (*prototype.TransactionReceiptWithInfo, error) {
	key, ok := d.accounts[name]
	if !ok {
		return nil, fmt.Errorf("unknown account: %s", name)
	}
	return d.SendTrxEx(key, operations...)
}

func (d *DandelionCore) TrxReceipt(privateKey *prototype.PrivateKeyType, operations...*prototype.Operation) *prototype.TransactionReceiptWithInfo {
	r, _ := d.SendTrxEx(privateKey, operations...)
	return r
}

func (d *DandelionCore) TrxReceiptByAccount(name string, operations...*prototype.Operation) *prototype.TransactionReceiptWithInfo {
	r, _ := d.SendTrxByAccountEx(name, operations...)
	return r
}

func (d *DandelionCore) ChainId() prototype.ChainId {
	return d.chainId
}

func (d *DandelionCore) beforePreShuffle() {
	for _, f := range d.beforePreshuffle {
		f()
	}
}

func (d *DandelionCore) afterPreShuffle() {
	for _, f := range d.afterPreshuffle {
		f()
	}
}

func (d *DandelionCore) SubscribePreShuffle(beforeOrAfter bool, f interface{}) {
	if cb, ok := f.(func()); ok {
		k := fmt.Sprintf("%v", f)
		if beforeOrAfter {
			d.beforePreshuffle[k] = cb
		} else {
			d.afterPreshuffle[k] = cb
		}
	}
}

func (d *DandelionCore) UnsubscribePreShuffle(beforeOrAfter bool, f interface{}) {
	if _, ok := f.(func()); ok {
		k := fmt.Sprintf("%v", f)
		if beforeOrAfter {
			delete(d.beforePreshuffle, k)
		} else {
			delete(d.afterPreshuffle, k)
		}
	}
}
