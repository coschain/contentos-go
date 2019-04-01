package plugins

import (
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/prototype"
	"github.com/sirupsen/logrus"
	"io/ioutil"
)

var FollowServiceName = "followsrv"

type FollowService struct {
	node.Service
	db  iservices.IDatabaseService
	log *logrus.Logger
	ev  EventBus.Bus
	ctx *node.ServiceContext
	pool iservices.ITrxPool
}

// service constructor
func NewFollowService(ctx *node.ServiceContext, lg *logrus.Logger) (*FollowService, error) {
	if lg == nil {
		lg = logrus.New()
		lg.SetOutput(ioutil.Discard)
	}
	return &FollowService{ctx:ctx, log:lg}, nil
}

func (p *FollowService) Start(node *node.Node) error {
	db, err := p.ctx.Service(iservices.DbServerName)
	if err != nil {
		return err
	}
	p.db = db.(iservices.IDatabaseService)
	pool, err := p.ctx.Service(iservices.TxPoolServerName)
	if err != nil {
		return err
	}
	p.pool = pool.(iservices.ITrxPool)
	p.ev = node.EvBus

	p.hookEvent()
	return nil
}

func (p *FollowService) hookEvent() {
	p.ev.Subscribe(constants.NoticeOpPost, p.onPostOperation)
}
func (p *FollowService) unhookEvent() {
	p.ev.Unsubscribe(constants.NoticeOpPost, p.onPostOperation)
}

func (p *FollowService) onPostOperation(notification *prototype.OperationNotification) {

	if notification.Op == nil {
		return
	}

	switch notification.Op.GetOp().(type) {
	case *prototype.Operation_Op8:
		//p.log.Debugf("receive follow operation [%x]", notification.Op.GetOp8())
		p.executeFollowOperation(notification.Op.GetOp8())
	default:

	}

}

func (p *FollowService) executeFollowOperation(op *prototype.FollowOperation) {
	/*
		FollowOperation{
			Account             A
			FAccount            B
			Cancel              bool
		}

		1. if Cancel == false, meaning A follow B
		2. if Cancel == true, meaning A cancel follow B
	*/

	currTime := p.pool.HeadBlockTime()

	// A's following
	fingWrap := table.NewSoExtFollowingWrap(p.db, &prototype.FollowingRelation{
		Account:   op.Account,
		Following: op.FAccount,
	})
	// B's follower
	ferWrap := table.NewSoExtFollowerWrap(p.db, &prototype.FollowerRelation{
		Account:  op.FAccount,
		Follower: op.Account,
	})
	// A's fing cnt
	fingCntWrap := table.NewSoExtFollowCountWrap(p.db, op.Account)
	// B's fer cnt
	ferCntWrap := table.NewSoExtFollowCountWrap(p.db, op.FAccount)

	if ferWrap == nil && fingWrap == nil && ferCntWrap == nil && fingCntWrap == nil {
		return
	}

	fingCnt := fingCntWrap.GetFollowingCnt()
	ferCnt := ferCntWrap.GetFollowerCnt()

	// add follow
	if !op.Cancel {
		if !fingWrap.CheckExist() {
			fingWrap.Create(func(fing *table.SoExtFollowing) {
				fing.FollowingInfo = &prototype.FollowingRelation{
					Account:   &prototype.AccountName{Value: op.Account.Value},
					Following: &prototype.AccountName{Value: op.FAccount.Value},
				}
				fing.FollowingCreatedOrder = &prototype.FollowingCreatedOrder{
					Account:     &prototype.AccountName{Value: op.Account.Value},
					CreatedTime: currTime,
					Following:   &prototype.AccountName{Value: op.FAccount.Value},
				}
			})

			ferWrap.Create(func(fer *table.SoExtFollower) {
				fer.FollowerInfo = &prototype.FollowerRelation{
					Account:  &prototype.AccountName{Value: op.FAccount.Value},
					Follower: &prototype.AccountName{Value: op.Account.Value},
				}
				fer.FollowerCreatedOrder = &prototype.FollowerCreatedOrder{
					Account:     &prototype.AccountName{Value: op.FAccount.Value},
					CreatedTime: currTime,
					Follower:    &prototype.AccountName{Value: op.Account.Value},
				}
			})

			//if fingCntWrap.CheckExist() {
			//	fingCntWrap.MdFollowingCnt(fingCnt + 1)
			//	ferCntWrap.MdFollowerCnt(ferCnt + 1)
			//	fmt.Printf("the account is %v ,follower count is %v, following count " +
			//		"is %v \n",ferCntWrap.GetAccount(),ferCntWrap.GetFollowerCnt(),ferCntWrap.GetFollowingCnt())
			//} else {
			//	fingCntWrap.Create(func(fCnt *table.SoExtFollowCount) {
			//		fCnt.Account = &prototype.AccountName{Value: op.Account.Value}
			//		fCnt.FollowingCnt = uint32(1)
			//		fCnt.FollowerCnt = uint32(0)
			//	})
			//
			//	ferCntWrap.Create(func(fCnt *table.SoExtFollowCount) {
			//		fCnt.Account = &prototype.AccountName{Value: op.FAccount.Value}
			//		fCnt.FollowingCnt = uint32(0)
			//		fCnt.FollowerCnt = uint32(1)
			//	})
			//}

			if fingCntWrap.CheckExist() {
				fingCntWrap.MdFollowingCnt(fingCnt + 1)
			}else {
				fingCntWrap.Create(func(fCnt *table.SoExtFollowCount) {
					fCnt.Account = &prototype.AccountName{Value: op.Account.Value}
					fCnt.FollowingCnt = uint32(1)
					fCnt.FollowerCnt = uint32(0)
				})
			}

			if ferCntWrap.CheckExist() {
				ferCntWrap.MdFollowerCnt(ferCnt + 1)
			}else {
				ferCntWrap.Create(func(fCnt *table.SoExtFollowCount) {
					fCnt.Account = &prototype.AccountName{Value: op.FAccount.Value}
					fCnt.FollowingCnt = uint32(0)
					fCnt.FollowerCnt = uint32(1)
				})
			}
		}
		// remove follow
	} else {
		if fingWrap.CheckExist() {
			fingWrap.RemoveExtFollowing()
			ferWrap.RemoveExtFollower()

			fingCntWrap.MdFollowingCnt(fingCnt - 1)
			ferCntWrap.MdFollowerCnt(ferCnt - 1)
		}
	}
}

func (p *FollowService) Stop() error {
	p.unhookEvent()
	return nil
}
