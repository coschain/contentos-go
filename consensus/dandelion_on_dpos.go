package consensus

import (
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/db/forkdb"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"github.com/sirupsen/logrus"
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
		ForkDB:         forkdb.NewDB(),
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

func (d *DPoS) DandelionDposSetController(ctrl iservices.ITrxPool) {
	d.ctrl = ctrl
}

func (d *DPoS) DandelionDposSetP2P(p2p iservices.IP2P) {
	d.p2p = p2p
}

func (d *DPoS) DandelionDposSetLog(log *logrus.Logger) {
	d.log = log
}

func (d *DPoS) DandelionDposOpenBlog(path string) {
	err := d.blog.Open(path)
	if err != nil {
		panic(err)
	}
}

func (d *DPoS) DandelionDposStart() {
	go d.testStart(snapshotPath)
}

func (d *DPoS) DandelionDposGenerateBlock(timestamp uint64) error {
	prev := &prototype.Sha256{}
	if !d.ForkDB.Empty() {
		prev.FromBlockID(d.ForkDB.Head().Id())
	} else {
		prev.Hash = make([]byte, 32)
	}
	b, err := d.ctrl.GenerateAndApplyBlock(d.Name, prev, uint32(timestamp), d.privKey, prototype.Skip_nothing)
	if err != nil {
		return err
	}
	err = d.pushBlock(b, true)
	if err != nil {
		return err
	}
	d.p2p.Broadcast(b)
	return nil
}

func (d *DPoS) DandelionDposStop() {
	d.stop(snapshotPath)
}
