package plugins

import (
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/prototype"
	"github.com/sirupsen/logrus"
	"time"
)

var TrxServiceName = "trxsrv"

type TrxService struct {
	node.Service
	db  iservices.IDatabaseService
	log *logrus.Logger
	ev  EventBus.Bus
	ctx *node.ServiceContext
}

func NewTrxSerVice(ctx *node.ServiceContext) (*TrxService,error) {
	return &TrxService{ctx:ctx}, nil
}

func (t *TrxService) Start(node *node.Node) error {
	db, err := t.ctx.Service(iservices.DbServerName)
	if err != nil {
		return err
	}
	t.db = db.(iservices.IDatabaseService)

	if err != nil {
		return err
	}
	t.ev = node.EvBus
	t.hookEvent()
	return nil
}

func (t *TrxService) hookEvent() {
	t.ev.Subscribe(constants.NoticeAddTrx, t.handleAddTrxNotification)
}
func (t *TrxService) unhookEvent() {
	t.ev.Unsubscribe(constants.NoticeAddTrx, t.handleAddTrxNotification)
}

func (t *TrxService) handleAddTrxNotification (blk *prototype.SignedBlock){
	if blk != nil && len(blk.Transactions) > 0 {
		count := uint64(len(blk.Transactions))
		now := time.Now()
		//Convert to time at 0 o'clock
		sTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		unixVal := sTime.Unix()
		wrap := table.NewSoExtDailyTrxWrap(t.db,&unixVal)
		//update daily total trx count
		if wrap != nil {
			if !wrap.CheckExist() {
				wrap.Create(func(tInfo *table.SoExtDailyTrx) {
					tInfo.Date = unixVal
					tInfo.Count = count
				})
			}else {
				curCnt := wrap.GetCount()
				wrap.MdCount(curCnt+count)
			}
		}

		//save trx info to db
		for _,trxWrap := range blk.Transactions {
			trxId,err :=  trxWrap.SigTrx.Id()
			if err == nil {
				wrap := table.NewSoExtTrxWrap(t.db,trxId)
				if wrap != nil {
					if !wrap.CheckExist() {
						 wrap.Create(func(tInfo *table.SoExtTrx) {
							tInfo.BlockTime = blk.SignedHeader.Header.Timestamp
							tInfo.BlockHeight = blk.Id().BlockNum()
							tInfo.TrxId = trxId
							tInfo.TrxWrap = trxWrap
						})
					}
				}
			}
		}
	}

}

func (t *TrxService) Stop() error {
	t.unhookEvent()
	return nil
}