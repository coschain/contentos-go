package prototype

type OperationNotification struct {
	trx_id       *Sha256
	block        uint32
	trx_in_block uint32
	op_in_trx    uint16
	virtual_op   uint64
	op           *Operation
}
