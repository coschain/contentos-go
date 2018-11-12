package prototype

type OperationNotification struct {
	Trx_id       *Sha256
	Block        uint32
	Trx_in_block uint32
	Op_in_trx    uint16
	Virtual_op   uint64
	Op           *Operation
}
