package dandelion

import (
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/app"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/consensus"
	"github.com/coschain/contentos-go/db/storage"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/mylog"
	"github.com/coschain/contentos-go/prototype"
)

const (
	blogPath     = "/tmp/blog"
	logPath      = "/tmp/log"
	snapshotPath = "/tmp/snapshot"
)

// for dpos
type RedDandelion struct {
	*consensus.DPoS
	*app.TrxPool
	path      string
	db        *storage.DatabaseService
	witness   string
	privKey   *prototype.PrivateKeyType
	produced  uint32
	timestamp uint64
	logger    iservices.ILog
}

func NewRedDandelion() (*RedDandelion, error) {
	db, err := storage.NewDatabase(dbPath)
	log, err := mylog.NewMyLog(logPath, "info", 0)
	if err != nil {
		log.GetLog().Error(err)
		return nil, err
	}
	if err != nil {
		log.GetLog().Error(err)
		return nil, err
	}
	privKey, err := prototype.PrivateKeyFromWIF(constants.INITMINER_PRIKEY)
	if err != nil {
		log.GetLog().Error(err)
		return nil, err
	}
	return &RedDandelion{path: dbPath, db: db, witness: "initminer", privKey: privKey, logger: log, timestamp: 3}, nil
}

func (d *RedDandelion) OpenDatabase() error {
	err := d.db.Start(nil)
	if err != nil {
		d.logger.GetLog().Error("open database error")
		return err
	}
	c, err := app.NewController(nil)
	if err != nil {
		d.logger.GetLog().Error("create new controller failed")
		return err
	}
	c.SetDB(d.db)
	c.SetBus(EventBus.New())
	c.Open()
	d.TrxPool = c
	p2p := NewDandelionP2P()
	dpos := consensus.NewDandelionDpos()
	dpos.DandelionDposSetController(c)
	dpos.DandelionDposSetP2P(p2p)
	dpos.DandelionDposOpenBlog(blogPath)
	dpos.DandelionDposSetLog(d.logger)
	dpos.DandelionDposStart()
	d.DPoS = dpos
	if err != nil {
		d.logger.GetLog().Error("dpos start error")
		return err
	}
	return nil
}

func (d *RedDandelion) GenerateBlock() {
	err := d.DPoS.DandelionDposGenerateBlock(d.timestamp)
	if err != nil {
		d.logger.GetLog().Error("error:", err)
	}
	d.produced += 1
	d.timestamp += constants.BLOCK_INTERVAL
}

func (d *RedDandelion) GenerateBlocks(count uint32) {
	for i := uint32(0); i < count; i++ {
		d.GenerateBlock()
	}
}

func (d *RedDandelion) GenerateBlockUntil(timestamp uint32) {
	if timestamp > d.GetProps().GetTime().UtcSeconds {
		count := (timestamp - d.GetProps().GetTime().UtcSeconds) / constants.BLOCK_INTERVAL
		d.GenerateBlocks(count)
	}
}

func (d *RedDandelion) GenerateBlockFor(timestamp uint32) {
	count := timestamp / constants.BLOCK_INTERVAL
	d.GenerateBlocks(count)
}

func (d *RedDandelion) CreateAccount(name string) error {
	defaultPrivKey, err := prototype.GenerateNewKeyFromBytes([]byte(initPrivKey))
	if err != nil {
		d.logger.GetLog().Error("error:", err)
		return err
	}
	defaultPubKey, err := defaultPrivKey.PubKey()
	if err != nil {
		d.logger.GetLog().Error("error:", err)
		return err
	}

	keys := prototype.NewAuthorityFromPubKey(defaultPubKey)

	// create account with default pub key
	acop := &prototype.AccountCreateOperation{
		Fee:            prototype.NewCoin(1),
		Creator:        &prototype.AccountName{Value: "initminer"},
		NewAccountName: &prototype.AccountName{Value: name},
		Owner:          keys,
	}
	// use initminer's priv key sign
	signTx, err := d.Sign(d.privKey.ToWIF(), acop)
	if err != nil {
		d.logger.GetLog().Error("error:", err)
		return err
	}
	d.DPoS.PushTransaction(signTx, true, true)
	d.GenerateBlock()
	return nil
}

func (d *RedDandelion) Transfer(from, to string, amount uint64, memo string) error {
	defaultPrivKey, err := prototype.GenerateNewKeyFromBytes([]byte(initPrivKey))
	if err != nil {
		d.logger.GetLog().Error("error:", err)
		return err
	}
	top := &prototype.TransferOperation{
		From:   &prototype.AccountName{Value: from},
		To:     &prototype.AccountName{Value: to},
		Amount: prototype.NewCoin(amount),
		Memo:   memo,
	}
	var signTx *prototype.SignedTransaction
	if from == "initminer" {
		signTx, err = d.Sign(d.privKey.ToWIF(), top)
	} else {
		signTx, err = d.Sign(defaultPrivKey.ToWIF(), top)
	}
	if err != nil {
		d.logger.GetLog().Error("error:", err)
		return err
	}
	d.DPoS.PushTransaction(signTx, true, true)
	d.GenerateBlock()
	return nil
}

func (d *RedDandelion) Fund(name string, amount uint64) error {
	return d.Transfer("initminer", name, amount, "")
}

func (d *RedDandelion) GetAccount(name string) *table.SoAccountWrap {
	accWrap := table.NewSoAccountWrap(d.db, &prototype.AccountName{Value: name})
	if !accWrap.CheckExist() {
		return nil
	}
	return accWrap
}

func (d *RedDandelion) Sign(privKeyStr string, ops ...interface{}) (*prototype.SignedTransaction, error) {
	privKey, err := prototype.PrivateKeyFromWIF(privKeyStr)
	if err != nil {
		return nil, err
	}
	props := d.TrxPool.GetProps()
	tx := &prototype.Transaction{RefBlockNum: 0, RefBlockPrefix: 0, Expiration: &prototype.TimePointSec{UtcSeconds: props.GetTime().UtcSeconds + constants.TRX_MAX_EXPIRATION_TIME}}
	headBlockID := props.GetHeadBlockId()
	id := &common.BlockID{}
	copy(id.Data[:], headBlockID.Hash[:])
	tx.SetReferenceBlock(id)
	for _, op := range ops {
		tx.AddOperation(op)
	}
	signTx := prototype.SignedTransaction{Trx: tx}
	res := signTx.Sign(privKey, prototype.ChainId{Value: 0})
	signTx.Signatures = append(signTx.Signatures, &prototype.SignatureType{Sig: res})
	if err := signTx.Validate(); err != nil {
		d.logger.GetLog().Error("error:", err)
		return nil, err
	}
	return &signTx, nil
}

func (d *RedDandelion) GetDB() *storage.DatabaseService {
	return d.db
}

func (d *RedDandelion) GetProduced() uint32 {
	return d.produced
}

func (d *RedDandelion) GeneralPrivKey() string {
	defaultPrivKey, _ := prototype.GenerateNewKeyFromBytes([]byte(initPrivKey))
	return defaultPrivKey.ToWIF()
}

func (d *RedDandelion) Clean() error {
	defer deletePath(d.path)
	defer deletePath(blogPath)
	defer deletePath(logPath)
	defer deletePath(snapshotPath)
	d.DandelionDposStop()
	err := d.db.Stop()
	if err != nil {
		d.logger.GetLog().Error("error:", err)
		return err
	}
	return nil
}
