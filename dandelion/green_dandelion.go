package dandelion

import (
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/app"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/db/storage"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"github.com/inconshreveable/log15"
	"os"
	"time"
)

const (
	dbPath = "/tmp/cos.db"
)

type GreenDandelion struct {
	iservices.IController
	path      string
	db        *storage.DatabaseService
	witness   string
	pre       *prototype.Sha256
	privKey   *prototype.PrivateKeyType
	timestamp uint32
	produced  uint32
	logger    log15.Logger
}

func NewDandelion(log log15.Logger) (*GreenDandelion, error) {
	db, err := storage.NewDatabase(dbPath)
	if err != nil {
		return nil, err
	}
	privKey, err := prototype.PrivateKeyFromWIF(constants.INITMINER_PRIKEY)
	if err != nil {
		return nil, err
	}
	pre := &prototype.Sha256{Hash: []byte{0}}
	return &GreenDandelion{path: dbPath, db: db, witness: constants.INIT_MINER_NAME, pre: pre, privKey: privKey,
		timestamp: 0, produced: 0, logger: log}, nil
}

func (d *GreenDandelion) OpenDatabase() error {
	err := d.db.Start(nil)
	if err != nil {
		d.logger.Error("open database error")
		return err
	}
	c, err := app.NewController(nil)
	if err != nil {
		d.logger.Error("create new controller failed")
	}
	c.SetDB(d.db)
	c.SetBus(EventBus.New())
	c.Open()
	d.IController = c
	return nil
}

func (d *GreenDandelion) GenerateBlock() {
	current := d.IController.GenerateBlock(d.witness, d.pre, d.timestamp, d.privKey, 0)
	d.timestamp += constants.BLOCK_INTERVAL
	currentHash := current.SignedHeader.Header.GetTransactionMerkleRoot()
	d.pre = currentHash
	d.produced += 1
}

func (d *GreenDandelion) Sign(ops ...interface{}) (*prototype.SignedTransaction, error) {
	tx := &prototype.Transaction{RefBlockNum: 0, RefBlockPrefix: 0, Expiration: &prototype.TimePointSec{UtcSeconds: uint32(time.Now().Unix()) + constants.TRX_MAX_EXPIRATION_TIME}}
	for _, op := range ops {
		tx.AddOperation(op)
	}
	signTx := prototype.SignedTransaction{Trx: tx}
	res := signTx.Sign(d.privKey, prototype.ChainId{Value: 0})
	signTx.Signatures = append(signTx.Signatures, &prototype.SignatureType{Sig: res})
	if err := signTx.Validate(); err != nil {
		return nil, err
	}
	return &signTx, nil
}

func (d *GreenDandelion) Clean() error {
	err := d.db.Stop()
	defer deleteDb(d.path)
	defer d.reset()
	if err != nil {
		return err
	}
	return nil
}

func (d *GreenDandelion) GetProduced() uint32 {
	return d.produced
}

func (d *GreenDandelion) GetTimestamp() uint32 {
	return d.timestamp
}

func (d *GreenDandelion) reset() {
	d.pre = &prototype.Sha256{Hash: []byte{0}}
	d.timestamp = 0
	d.privKey = nil
	d.produced = 0
	d.witness = ""
}

func deleteDb(path string) {
	_ = os.Remove(path)
}
