package dandelion

import (
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/app"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/db/storage"
	"github.com/coschain/contentos-go/prototype"
	"github.com/inconshreveable/log15"
	"os"
)

const (
	dbPath = "/tmp/cos.db"
)

type GreenDandelion struct {
	*app.Controller
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
		log.Error("error:", err)
		return nil, err
	}
	privKey, err := prototype.PrivateKeyFromWIF(constants.INITMINER_PRIKEY)
	if err != nil {
		log.Error("error:", err)
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
	d.Controller = c
	d.timestamp = c.GetProps().GetTime().UtcSeconds
	return nil
}

func (d *GreenDandelion) SetWitness(name string, privKey *prototype.PrivateKeyType) {
	d.witness = name
	d.privKey = privKey
}

func (d *GreenDandelion) GenerateBlock() {
	current := d.Controller.GenerateBlock(d.witness, d.pre, d.timestamp, d.privKey, 0)
	d.timestamp += constants.BLOCK_INTERVAL
	currentHash := current.SignedHeader.Header.GetTransactionMerkleRoot()
	d.pre = currentHash
	d.produced += 1
	err := d.PushBlock(current, prototype.Skip_nothing)
	if err != nil {
		d.logger.Error("error", err)
	}
}

func (d *GreenDandelion) GenerateBlocks(count uint32) {
	for i := uint32(0); i < count; i++ {
		d.GenerateBlock()
	}
}

func (d *GreenDandelion) GenerateBlockUntil(timestamp uint32) {
	count := (timestamp - d.GetProps().GetTime().UtcSeconds) / constants.BLOCK_INTERVAL
	d.GenerateBlocks(count)
}

func (d *GreenDandelion) GenerateBlockFor(timestamp uint32) {
	count := timestamp / constants.BLOCK_INTERVAL
	d.GenerateBlocks(count)
}

func (d *GreenDandelion) Sign(ops ...interface{}) (*prototype.SignedTransaction, error) {
	props := d.Controller.GetProps()
	tx := &prototype.Transaction{RefBlockNum: 0, RefBlockPrefix: 0, Expiration: &prototype.TimePointSec{UtcSeconds: props.GetTime().UtcSeconds + constants.TRX_MAX_EXPIRATION_TIME}}
	headBlockID := props.GetHeadBlockId()
	id := &common.BlockID{}
	copy(id.Data[:], headBlockID.Hash[:])
	tx.SetReferenceBlock(id)
	for _, op := range ops {
		tx.AddOperation(op)
	}
	signTx := prototype.SignedTransaction{Trx: tx}
	res := signTx.Sign(d.privKey, prototype.ChainId{Value: 0})
	signTx.Signatures = append(signTx.Signatures, &prototype.SignatureType{Sig: res})
	if err := signTx.Validate(); err != nil {
		d.logger.Error("error:", err)
		return nil, err
	}
	return &signTx, nil
}

func (d *GreenDandelion) CreateAccount(name string) error {
	acop := &prototype.AccountCreateOperation{
		Fee:            prototype.NewCoin(1),
		Creator:        &prototype.AccountName{Value: "initminer"},
		NewAccountName: &prototype.AccountName{Value: name},
		Owner: &prototype.Authority{
			Cf:              prototype.Authority_owner,
			WeightThreshold: 1,
			AccountAuths: []*prototype.KvAccountAuth{
				&prototype.KvAccountAuth{
					Name:   &prototype.AccountName{Value: "initminer"},
					Weight: 3,
				},
			},
			KeyAuths: []*prototype.KvKeyAuth{
				&prototype.KvKeyAuth{
					Key: &prototype.PublicKeyType{
						Data: []byte{0},
					},
					Weight: 23,
				},
			},
		},
		Active:  &prototype.Authority{},
		Posting: &prototype.Authority{},
	}
	signTx, err := d.Sign(acop)
	if err != nil {
		d.logger.Error("error:", err)
		return err
	}
	d.PushTrx(signTx)
	d.GenerateBlock()
	return nil
}

func (d *GreenDandelion) Transfer(from, to string, amount uint64, memo string) error {
	top := &prototype.TransferOperation{
		From:   &prototype.AccountName{Value: from},
		To:     &prototype.AccountName{Value: to},
		Amount: prototype.NewCoin(amount),
		Memo:   memo,
	}
	signTx, err := d.Sign(top)
	if err != nil {
		d.logger.Error("error:", err)
		return err
	}
	d.PushTrx(signTx)
	d.GenerateBlock()
	return nil
}

func (d *GreenDandelion) Fund(name string, amount uint64) error {
	return d.Transfer("initminer", name, amount, "")
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

func (d *GreenDandelion) GetAccount(name string) *table.SoAccountWrap {
	accWrap := table.NewSoAccountWrap(d.db, &prototype.AccountName{Value: name})
	if !accWrap.CheckExist() {
		return nil
	}
	return accWrap
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
