package prototype

func (m *Transaction) set_expiration(time int) {
}

func (m *Transaction) AddOperation(op interface{}) {

	res := &Operation{}
	switch op.(type) {
	case *AccountCreateOperation:
		ptr := op.(*AccountCreateOperation)
		res.Op = &Operation_Op1{Op1: ptr}
		break
	case *TransferOperation:
		ptr := op.(*TransferOperation)
		res.Op = &Operation_Op2{Op2: ptr}
		break
	default:
		panic("error op type")
	}
	m.Operations = append(m.Operations, res)
}