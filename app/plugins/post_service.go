package plugins

import (
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/common/logging"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/prototype"
)

var POST_SERVICE_NAME = "postsrv"

type PostService struct {
	node.Service
	db iservices.IDatabaseService
	ev EventBus.Bus
	ctx *node.ServiceContext
}

// service constructor
func NewPostService(ctx *node.ServiceContext) (*PostService, error) {
	return &PostService{ctx: ctx}, nil
}


func (p *PostService) Start(node *node.Node) error {
	db, err := p.ctx.Service(iservices.DB_SERVER_NAME)
	if err != nil {
		return err
	}
	p.db = db.(iservices.IDatabaseService)
	p.ev = node.EvBus

	p.hookEvent()
	return nil
}

func (p *PostService) hookEvent() {
	p.ev.Subscribe( constants.NOTICE_OP_POST , p.onPostOperation )
}
func (p *PostService) unhookEvent() {
	p.ev.Unsubscribe( constants.NOTICE_OP_POST , p.onPostOperation )
}

func (p *PostService) onPostOperation( notification *prototype.OperationNotification )  {

	if notification.Op == nil || notification.Op.GetOp6() == nil {
		return
	}

	op := notification.Op.GetOp6()
	logging.CLog().Infof("receive post-op event: %v", op)
	// TODO
}


func (p *PostService) Stop() error{
	p.unhookEvent()
	return nil
}
