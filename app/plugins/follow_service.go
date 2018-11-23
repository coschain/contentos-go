package plugins

import (
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/prototype"
)

var FOLLOW_SERVICE_NAME = "followsrv"

type FollowService struct {
	node.Service
	db iservices.IDatabaseService
	ev EventBus.Bus
	ctx *node.ServiceContext
}

// service constructor
func NewFollowService(ctx *node.ServiceContext) (*FollowService, error) {
	return &FollowService{ctx: ctx}, nil
}


func (p *FollowService) Start(node *node.Node) error {
	db, err := p.ctx.Service(iservices.DB_SERVER_NAME)
	if err != nil {
		return err
	}
	p.db = db.(iservices.IDatabaseService)
	p.ev = node.EvBus

	p.hookEvent()
	return nil
}

func (p *FollowService) hookEvent() {
	p.ev.Subscribe( constants.NOTICE_OP_POST , p.onPostOperation )
}
func (p *FollowService) unhookEvent() {
	p.ev.Unsubscribe( constants.NOTICE_OP_POST , p.onPostOperation )
}

func (p *FollowService) onPostOperation( notification *prototype.OperationNotification )  {

	if notification.Op == nil || notification.Op.GetOp8() == nil {
		return
	}

	op := notification.Op.GetOp8()

	followerWrap := table.NewSoExtFollowCountWrap( p.db, op.Follower )

	// TODO update follow data
	if followerWrap.CheckExist() {

	}
}


func (p *FollowService) Stop() error{
	p.unhookEvent()
	return nil
}
