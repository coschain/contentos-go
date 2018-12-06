package consensus

import (
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/db/forkdb"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"time"
)

const (
	snapshotPath = "/tmp/snapshot"
)

func NewDandelionDpos() *DPoS {
	privKey, err := prototype.PrivateKeyFromWIF(constants.INITMINER_PRIKEY)
	if err != nil {
		panic("initminer privkey parser error")
	}
	ret := &DPoS{
		ForkDB:         forkdb.NewDB(nil),
		Name:           "initminer",
		Producers:      []string{"initminer"},
		bootstrap:      true,
		readyToProduce: true,
		prodTimer:      time.NewTimer(86400 * time.Second),
		privKey:        privKey,
		trxCh:          make(chan func()),
		blkCh:          make(chan common.ISignedBlock),
		stopCh:         make(chan struct{}),
	}
	return ret
}

func (d *DPoS) DandelionDposSetController(ctrl iservices.IController) {
	d.ctrl = ctrl
}

func (d *DPoS) DandelionDposSetP2P(p2p iservices.IP2P) {
	d.p2p = p2p
}

func (d *DPoS) DandelionDposSetLog(log iservices.ILog) {
	d.log = log
}

func (d *DPoS) DandelionDposOpenBlog(path string) {
	err := d.blog.Open(path)
	if err != nil {
		panic(err)
	}
}

func (d *DPoS) DandelionDposStart() {
	go d.start(snapshotPath)
}

func (d *DPoS) DandelionDposGenerateBlock() error {
	b, err := d.generateAndApplyBlock()
	if err != nil {
		return err
	}
	err = d.pushBlock(b, false)
	if err != nil {
		return err
	}
	d.p2p.Broadcast(b)
	return nil
}

func (d *DPoS) DandelionDposStop() {
	d.stop(snapshotPath)
}
