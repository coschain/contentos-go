package core

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/app"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/common/eventloop"
	"github.com/coschain/contentos-go/config"
	"github.com/coschain/contentos-go/db/storage"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/prototype"
	"github.com/gogo/protobuf/proto"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

type DandelionCore struct {
	node *node.Node
	cfg node.Config
	chainId prototype.ChainId
	timeStamp uint32
	prevHash *prototype.Sha256
	accounts map[string]*prototype.PrivateKeyType
}

func NewDandelionCore(logger *logrus.Logger) *DandelionCore {
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

	_ = n.Register(iservices.DbServerName, func(ctx *node.ServiceContext) (node.Service, error) {
		return storage.NewGuardedDatabaseService(ctx, "./db/")
	})
	_ = n.Register(iservices.TxPoolServerName, func(ctx *node.ServiceContext) (node.Service, error) {
		return app.NewController(ctx, n.Log)
	})
	_ = n.Register(DummyConsensusName, func(ctx *node.ServiceContext) (node.Service, error) {
		return NewDummyConsensus(ctx)
	})

	d := &DandelionCore{
		node: n,
		cfg: cfg,
		chainId: prototype.ChainId{ Value: common.GetChainIdByName(cfg.ChainId) },
		timeStamp: uint32(time.Now().Unix()),
		prevHash: &prototype.Sha256{ Hash: make([]byte, 32) },
		accounts: make(map[string]*prototype.PrivateKeyType),
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
	}
	return
}

func (d *DandelionCore) Stop() error {
	defer d.cleanup()
	return d.node.Stop()
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

func (d *DandelionCore) ProduceBlocks(count int) error {
	const skip = prototype.Skip_block_signatures
	var (
		block common.ISignedBlock
		blockId common.BlockID
		err error
	)
	copy(blockId.Data[:], d.prevHash.Hash)
	num := blockId.BlockNum() + 1
	for i := 0; i < count; i++ {
		bp := d.Consensus().GetProducer(num)
		bpKey, ok := d.accounts[bp]
		if !ok {
			return fmt.Errorf("unknown block producer: %s", bp)
		}
		block, err = d.TrxPool().GenerateAndApplyBlock(bp, d.prevHash, d.timeStamp, bpKey, skip)
		if err != nil {
			break
		}
		blockId = block.Id()
		d.TrxPool().Commit(num)
		copy(d.prevHash.Hash, blockId.Data[:])
		d.timeStamp += constants.BlockInterval
		num++
	}
	return err
}

func (d *DandelionCore) SendTrx(privateKey *prototype.PrivateKeyType, operations...*prototype.Operation) error {
	data, err := proto.Marshal(&prototype.Transaction{
		RefBlockNum: common.TaposRefBlockNum(d.Head().BlockNum()),
		RefBlockPrefix: common.TaposRefBlockPrefix(d.prevHash.Hash),
		Expiration: prototype.NewTimePointSec(d.timeStamp + constants.TrxMaxExpirationTime - 1),
		Operations: operations,
	},)
	if err != nil {
		return err
	}
	trx := new(prototype.Transaction)
	if err = proto.Unmarshal(data, trx); err != nil {
		return err
	}
	signedTrx := &prototype.SignedTransaction{
		Trx: trx,
		Signature: new(prototype.SignatureType),
	}
	signedTrx.Signature.Sig = signedTrx.Sign(privateKey, d.chainId)
	return d.TrxPool().PushTrxToPending(signedTrx)
}

func (d *DandelionCore) SendTrxByAccount(name string, operations...*prototype.Operation) error {
	key, ok := d.accounts[name]
	if !ok {
		return fmt.Errorf("unknown account: %s", name)
	}
	return d.SendTrx(key, operations...)
}
