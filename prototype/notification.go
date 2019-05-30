package prototype

type OperationNotification struct {
	Trx_id       *Sha256
	Trx_status   uint32
	Block        uint64
	Trx_in_block uint64
	Op_in_trx    uint64
	Virtual_op   uint64
	Op           *Operation
}
