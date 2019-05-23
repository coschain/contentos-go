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

func NewTrxSerVice(ctx *node.ServiceContext, log *logrus.Logger) (*TrxService, error) {
	return &TrxService{ctx: ctx, log: log}, nil
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
	t.ev.Subscribe(constants.NoticeBlockApplied, t.handleAddTrxNotification)
}
func (t *TrxService) unhookEvent() {
	t.ev.Unsubscribe(constants.NoticeBlockApplied, t.handleAddTrxNotification)
}

func (t *TrxService) handleAddTrxNotification(blk *prototype.SignedBlock) {
	if blk != nil && len(blk.Transactions) > 0 {
		t0 := time.Now()

		count := uint64(len(blk.Transactions))

		timestamp := blk.SignedHeader.Header.Timestamp
		index := timestamp.UtcSeconds / 86400
		hourIndex := timestamp.UtcSeconds / 3600
		wrap := table.NewSoExtDailyTrxWrap(t.db, &prototype.TimePointSec{UtcSeconds: index})
		//update daily total trx count
		if wrap != nil {
			if !wrap.CheckExist() {
				wrap.Create(func(tInfo *table.SoExtDailyTrx) {
					tInfo.Date = &prototype.TimePointSec{UtcSeconds: index}
					tInfo.Count = count
				})
			} else {
				curCnt := wrap.GetCount()
				wrap.Md(func(tInfo *table.SoExtDailyTrx) {
					tInfo.Count = curCnt + count
				})
			}
		}
		hourwrap := table.NewSoExtHourTrxWrap(t.db, &prototype.TimePointSec{UtcSeconds: hourIndex})
		if hourwrap != nil {
			if !hourwrap.CheckExist() {
				hourwrap.Create(func(tInfo *table.SoExtHourTrx) {
					tInfo.Hour = &prototype.TimePointSec{UtcSeconds: hourIndex}
					tInfo.Count = count
				})
			} else {
				curCnt := hourwrap.GetCount()
				hourwrap.Md(func(tInfo *table.SoExtHourTrx) {
					tInfo.Count = curCnt + count
				})
			}
		}

		t1 := time.Now()
		//save trx info to db
		for _, trxWrap := range blk.Transactions {
			trxId, err := trxWrap.SigTrx.Id()
			if err == nil {
				wrap := table.NewSoExtTrxWrap(t.db, trxId)
				if wrap != nil {
					if !wrap.CheckExist() {
						creator := t.GetTrxCreator(trxWrap.SigTrx.GetOpCreatorsMap())
						creAcct := prototype.NewAccountName(creator)
						wrap.Create(func(tInfo *table.SoExtTrx) {
							tInfo.BlockTime = blk.SignedHeader.Header.Timestamp
							tInfo.BlockHeight = blk.Id().BlockNum()
							tInfo.TrxId = trxId
							tInfo.TrxWrap = trxWrap
							tInfo.TrxCreateOrder = &prototype.UserTrxCreateOrder{
								Creator:creAcct,
								CreateTime:tInfo.BlockTime,
							}

							bId := blk.Id().Data
							tInfo.BlockId = &prototype.Sha256{Hash:bId[:]}
						})
						//update user's created trx count
						acctWrap := table.NewSoAccountWrap(t.db,creAcct)
						if acctWrap != nil && acctWrap.CheckExist() {
							curCnt := acctWrap.GetCreatedTrxCount()
							acctWrap.Md(func(tInfo *table.SoAccount) {
								tInfo.CreatedTrxCount = curCnt+1
							})
						}
					}
				}
			}
		}
		t2 := time.Now()
		t.log.Debugf("TXSVC: %v|%v|%v", t2.Sub(t0), t1.Sub(t0), t2.Sub(t1))
	}

}

func (t *TrxService) GetTrxCreator(usrMap map[string]bool) string {
	if len(usrMap) > 0 {
		for k := range usrMap {
			return k
		}
	}
	return ""
}

func (t *TrxService) Stop() error {
	t.unhookEvent()
	return nil
}
