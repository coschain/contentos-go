package plugins

import (
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/prototype"
)

var PostServiceName = "postsrv"

type PostService struct {
	node.Service
	db  iservices.IDatabaseService
	ev  EventBus.Bus
	pool iservices.ITrxPool
	ctx *node.ServiceContext
}

// service constructor
func NewPostService(ctx *node.ServiceContext) (*PostService, error) {
	return &PostService{ctx:ctx}, nil
}

func (p *PostService) Start(node *node.Node) error {
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

func (p *PostService) hookEvent() {
	p.ev.Subscribe(constants.NoticeOpPost, p.onPostOperation)
}
func (p *PostService) unhookEvent() {
	p.ev.Unsubscribe(constants.NoticeOpPost, p.onPostOperation)
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


	createTime := p.pool.HeadBlockTime()
	exPostWrap := table.NewSoExtPostCreatedWrap(p.db, &uuid)
	if exPostWrap != nil && !exPostWrap.CheckExist() {
		exPostWrap.Create(func(exPost *table.SoExtPostCreated) {
			exPost.PostId = uuid
			exPost.CreatedOrder = &prototype.PostCreatedOrder{
				Created: createTime,
				ParentId: constants.PostInvalidId,
			}
		})
	}
	userPostWrap := table.NewSoExtUserPostWrap(p.db,&op.Uuid)
	if userPostWrap != nil && !userPostWrap.CheckExist() {
		 userPostWrap.Create(func(tInfo *table.SoExtUserPost) {
			tInfo.PostId = op.Uuid
			tInfo.PostCreatedOrder = &prototype.UserPostCreateOrder{
				Author:op.Owner,
				Create:createTime,
			}
		})

	}
}

func (p *PostService) executeReplyOperation(op *prototype.ReplyOperation) {
	uuid := op.GetUuid()
	exReplyWrap := table.NewSoExtReplyCreatedWrap(p.db, &uuid)
	if exReplyWrap != nil && !exReplyWrap.CheckExist() {
		exReplyWrap.Create(func(exReply *table.SoExtReplyCreated) {
			exReply.PostId = uuid
			exReply.CreatedOrder = &prototype.ReplyCreatedOrder{
				ParentId: op.GetParentUuid(),
				Created: p.pool.HeadBlockTime(),
			}
		})
	}
}

func (p *PostService) Stop() error {
	p.unhookEvent()
	return nil
}
