package dandelion

import (
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/app"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/consensus"
	"github.com/coschain/contentos-go/db/storage"
	"github.com/coschain/contentos-go/prototype"
	"github.com/inconshreveable/log15"
)

const (
	blogPath = "/tmp/blog"
)

// for dpos
type RedDandelion struct {
	*consensus.DPoS
	path    string
	db      *storage.DatabaseService
	witness string
	privKey *prototype.PrivateKeyType
	logger  log15.Logger
}

func NewRedDandelion() (*RedDandelion, error) {
	db, err := storage.NewDatabase(dbPath)
	log := log15.New()
	if err != nil {
		log.Error("error:", err)
		return nil, err
	}
	privKey, err := prototype.PrivateKeyFromWIF(constants.INITMINER_PRIKEY)
	if err != nil {
		log.Error("error:", err)
		return nil, err
	}
	return &RedDandelion{path: dbPath, db: db, witness: "initminer", privKey: privKey, logger: log}, nil
}

func (d *RedDandelion) OpenDatabase() error {
	err := d.db.Start(nil)
	if err != nil {
		d.logger.Error("open database error")
		return err
	}
	c, err := app.NewController(nil)
	if err != nil {
		d.logger.Error("create new controller failed")
		return err
	}
	c.SetDB(d.db)
	c.SetBus(EventBus.New())
	c.Open()
	p2p := NewDandelionP2P()
	dpos := consensus.NewDandelionDpos()
	dpos.DandelionDposSetController(c)
	dpos.DandelionDposSetP2P(p2p)
	dpos.DandelionDposOpenBlog(blogPath)
	err = dpos.Start(nil)
	d.DPoS = dpos
	if err != nil {
		d.logger.Error("dpos start error")
		return err
	}
	return nil
}

func (d *RedDandelion) GenerateBlock() {
	err := d.DPoS.DandelionDposGenerateBlock()
	d.logger.Error("error:", err)
}

func (d *RedDandelion) Clean() error {
	d.DandelionDposStop()
	err := d.db.Stop()
	defer deletePath(d.path)
	defer deletePath(blogPath)
	if err != nil {
		d.logger.Error("clean err", err)
		return err
	}
	return nil
}
