package app

import (
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/proto/type-proto"
)

type Controller struct {
	// lock for db write
	// pending_trx_list
	// DB Manager
	db      *AppDBLayer
	noticer EventBus.Bus
}

func (c *Controller) Start() {

}

func (c *Controller) Stop() {

}

func (c *Controller) PushTrx(trx *prototype.SignedTransaction) {

}

func (c *Controller) PushBlock(blk *prototype.SignedBlock) {

}

func (c *Controller) GenerateBlock(key *prototype.PrivateKeyType) *prototype.SignedBlock {
	return nil
}

func (c *Controller) NotifyOpPostExecute(on *prototype.OperationNotification) {
	c.noticer.Publish(constants.NOTICE_OP_POST, *on)
}

func (c *Controller) NotifyOpPreExecute(on *prototype.OperationNotification) {
	c.noticer.Publish(constants.NOTICE_OP_PRE, *on)
}

func (c *Controller) NotifyTrxPreExecute(trx *prototype.SignedTransaction) {
	c.noticer.Publish(constants.NOTICE_TRX_PRE, *trx)
}

func (c *Controller) NotifyTrxPostExecute(trx *prototype.SignedTransaction) {
	c.noticer.Publish(constants.NOTICE_TRX_POST, *trx)
}

// calculate reward for creator and witness
func (c *Controller) processBlock() {
}
