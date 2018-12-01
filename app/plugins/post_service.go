package plugins

import (
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/prototype"
)

var POST_SERVICE_NAME = "postsrv"

type PostService struct {
	node.Service
	db  iservices.IDatabaseService
	ev  EventBus.Bus
	ctx *node.ServiceContext
}

// service constructor
func NewPostService(ctx *node.ServiceContext) (*PostService, error) {
	return &PostService{ctx: ctx}, nil
}

func (p *PostService) Start(node *node.Node) error {
	db, err := p.ctx.Service(iservices.DbServerName)
	if err != nil {
		return err
	}
	p.db = db.(iservices.IDatabaseService)
	p.ev = node.EvBus

	p.hookEvent()
	return nil
}

func (p *PostService) hookEvent() {
	p.ev.Subscribe(constants.NOTICE_OP_POST, p.onPostOperation)
}
func (p *PostService) unhookEvent() {
	p.ev.Unsubscribe(constants.NOTICE_OP_POST, p.onPostOperation)
}

func (p *PostService) onPostOperation(notification *prototype.OperationNotification) {

	if notification.Op == nil {
		return
	}

	switch notification.Op.GetOp().(type) {
	case *prototype.Operation_Op6:
		p.executePostOperation(notification.Op.GetOp6())
	case *prototype.Operation_Op7:
		p.executeReplyOperation(notification.Op.GetOp7())
	default:

	}
}

func (p *PostService) executePostOperation(op *prototype.PostOperation) {
	uuid := op.GetUuid()
	ctrl, err := p.ctx.Service(iservices.CTRL_SERVER_NAME)
	if err != nil {
		panic("ctrl service invalid")
	}

	exPostWrap := table.NewSoExtPostCreatedWrap(p.db, &uuid)
	if exPostWrap != nil && !exPostWrap.CheckExist() {
		exPostWrap.Create(func(exPost *table.SoExtPostCreated) {
			exPost.PostId = uuid
			exPost.CreatedOrder = &prototype.PostCreatedOrder{
				Created: ctrl.(iservices.IController).HeadBlockTime(),
				ParentId: constants.POST_INVALID_ID,
			}
		})
	}
}

func (p *PostService) executeReplyOperation(op *prototype.ReplyOperation) {
	uuid := op.GetUuid()
	ctrl, err := p.ctx.Service(iservices.CTRL_SERVER_NAME)
	if err != nil {
		panic("ctrl service invalid")
	}
	exReplyWrap := table.NewSoExtReplyCreatedWrap(p.db, &uuid)
	if exReplyWrap != nil && !exReplyWrap.CheckExist() {
		exReplyWrap.Create(func(exReply *table.SoExtReplyCreated) {
			exReply.PostId = uuid
			exReply.CreatedOrder = &prototype.ReplyCreatedOrder{
				ParentId: op.GetParentUuid(),
				Created: ctrl.(iservices.IController).HeadBlockTime(),
			}
		})
	}
}

func (p *PostService) Stop() error {
	p.unhookEvent()
	return nil
}
