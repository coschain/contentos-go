package prototype

import (
	"crypto/sha256"
	"github.com/gogo/protobuf/proto"
)

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

func (m *Transaction) Hash() (hash [Size]byte) {
	data, _ := proto.Marshal(m)
	hash = sha256.Sum256(data)
	return
}