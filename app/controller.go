package app

import (
	"github.com/coschain/contentos-go/proto/type-proto"
	"github.com/coschain/contentos-go/proto/common-interface"
)

type Controller struct {
	// lock for db write
	// pending_trx_list
	// DB Manager
	db *AppDBLayer
}

func (c *Controller) Start()  {

}

func (c *Controller) Stop()  {

}


func (c *Controller) PushTrx( trx * prototype.SignedTransaction )  {

}

func (c *Controller) PushBlock( blk * prototype.SignedBlock )  {

}

func (c *Controller) GenerateBlock( key *prototype.PrivateKeyType) *prototype.SignedBlock  {
	return nil
}


func (c *Controller) NotifyOpPostExecute( op *commoninterface.BaseOperation) {
}

func (c *Controller) NotifyOpPreExecute( op *commoninterface.BaseOperation) {
}

func (c *Controller) NotifyTrxPreExecute( trx *prototype.SignedTransaction) {
}
func (c *Controller) NotifyTrxPostExecute( trx *prototype.SignedTransaction) {
}

// calculate reward for creator and witness
func (c *Controller) processBlock() {
}