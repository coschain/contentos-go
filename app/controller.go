package app

import (
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/proto/type-proto"
)

type skipFlag uint32

const (
	skip_nothing                skipFlag = 0
	skip_transaction_signatures skipFlag = 1 << 0
	skip_apply_transaction      skipFlag = 1 << 1
)

type Controller struct {
	// lock for db write
	// pending_trx_list
	// DB Manager
	db      *AppDBLayer
	noticer EventBus.Bus
	skip    skipFlag

	_pending_tx   []*prototype.TransactionWrapper
	_isProducing  bool
	_currentTrxId *prototype.Sha256
}

func (c *Controller) Start() {

}

func (c *Controller) Stop() {

}

func (c *Controller) setProducing(b bool) {
	c._isProducing = b
}

func (c *Controller) PushTrx(trx *prototype.SignedTransaction) *prototype.TransactionInvoice {
	// this function may be cross routines ? use channel or lock ?
	defer func() {
		c.setProducing(false)
	}()

	// @ check maximum_block_size

	c.setProducing(true)
	return c._pushTrx(trx)
}

func (c *Controller) _pushTrx(trx *prototype.SignedTransaction) *prototype.TransactionInvoice {
	defer func() {
		// @ undo sub session
	}()
	// @ start a new undo session when first transaction come after push block

	trxWrp := &prototype.TransactionWrapper{}
	trxWrp.SigTrx = trx

	// @ start a sub undo session for applyTransaction

	c._applyTransaction(trxWrp)
	c._pending_tx = append(c._pending_tx, trxWrp)

	// @ commit sub session

	c.NotifyTrxPending(trx)
	return trxWrp.Invoice
}

func (c *Controller) PushBlock(blk *prototype.SignedBlock) {

}

func (c *Controller) GenerateBlock(key *prototype.PrivateKeyType) *prototype.SignedBlock {
	return nil
}

func (c *Controller) NotifyOpPostExecute(on *prototype.OperationNotification) {
	c.noticer.Publish(constants.NOTICE_OP_POST, on)
}

func (c *Controller) NotifyOpPreExecute(on *prototype.OperationNotification) {
	c.noticer.Publish(constants.NOTICE_OP_PRE, on)
}

func (c *Controller) NotifyTrxPreExecute(trx *prototype.SignedTransaction) {
	c.noticer.Publish(constants.NOTICE_TRX_PRE, trx)
}

func (c *Controller) NotifyTrxPostExecute(trx *prototype.SignedTransaction) {
	c.noticer.Publish(constants.NOTICE_TRX_POST, trx)
}

func (c *Controller) NotifyTrxPending(trx *prototype.SignedTransaction) {
	c.noticer.Publish(constants.NOTICE_TRX_PENDING, trx)
}

func (c *Controller) NotifyBlockApply(block *prototype.SignedBlock) {
	c.noticer.Publish(constants.NOTICE_BLOCK_APPLY, block)
}

// calculate reward for creator and witness
func (c *Controller) processBlock() {
}

func (c *Controller) _applyTransaction(trxWrp *prototype.TransactionWrapper) {
	trx := trxWrp.SigTrx
	var err error
	c._currentTrxId, err = trx.Id()
	if err != nil {
	}

	trx.Validate()

	// @ trx duplicate check

	if c.skip&skip_transaction_signatures == 0 {

	}
}
