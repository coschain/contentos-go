package plugins

import (
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/common/logging"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/prototype"
)
var DEMO_SERVICE_NAME = "demosrv"

type DemoService struct {
	node.Service
	db iservices.IDatabaseService
	ev EventBus.Bus
	ctx *node.ServiceContext
}

// service constructor
func NewDemoService(ctx *node.ServiceContext) (*DemoService, error) {
	return &DemoService{ctx: ctx}, nil
}


func (p *DemoService) Start(node *node.Node) error {
	db, err := p.ctx.Service(iservices.DB_SERVER_NAME)
	if err != nil {
		return err
	}
	p.db = db.(iservices.IDatabaseService)
	p.ev = node.EvBus

	p.hookEvent()
	return nil
}

func (p *DemoService) hookEvent() {
	p.ev.Subscribe( constants.NOTICE_OP_POST , p.onPostOperation )
}
func (p *DemoService) unhookEvent() {
	p.ev.Unsubscribe( constants.NOTICE_OP_POST , p.onPostOperation )
}

func (p *DemoService) onPostOperation( notification *prototype.OperationNotification )  {
	// TODO add handle code
	logging.CLog().Infof("onPostOperation: %v", notification.Op )
}


func (p *DemoService) Stop() error{
	p.unhookEvent()
	return nil
}
