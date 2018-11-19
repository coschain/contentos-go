package prototype

import "errors"

func (m *Transaction) set_expiration(time int) {
}

func (m *Transaction) Validate() error {
	if m == nil {
		return ErrNpe
	}

	if m.Expiration == nil {
		return errors.New("trx must has Expiration")
	}

	if m.Operations == nil || len(m.Operations) == 0 {
		return errors.New("trx must has Operations")
	}

	for _,op := range m.Operations {
		if err := validateOp(op); err != nil{
			return err
		}
	}

	return nil
}

func validateOp(op *Operation)  error {
	if op == nil {
		return ErrNpe
	}

	if op.GetOp1() != nil {
		return op.GetOp1().Validate()
	}
	if op.GetOp2() != nil {
		return op.GetOp2().Validate()
	}
	if op.GetOp3() != nil {
		return op.GetOp3().Validate()
	}
	if op.GetOp4() != nil {
		return op.GetOp4().Validate()
	}
	if op.GetOp5() != nil {
		return op.GetOp5().Validate()
	}
	if op.GetOp6() != nil {
		return op.GetOp6().Validate()
	}
	if op.GetOp7() != nil {
		return op.GetOp7().Validate()
	}
	if op.GetOp8() != nil {
		return op.GetOp8().Validate()
	}
	if op.GetOp9() != nil {
		return op.GetOp9().Validate()
	}
	if op.GetOp9() != nil {
		return op.GetOp9().Validate()
	}
	if op.GetOp10() != nil {
		return op.GetOp10().Validate()
	}

	return errors.New("unknown op type")
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
	case *BpRegisterOperation:
		ptr := op.(*BpRegisterOperation)
		res.Op = &Operation_Op3{Op3: ptr}
		break
	case *BpUnregisterOperation:
		ptr := op.(*BpUnregisterOperation)
		res.Op = &Operation_Op4{Op4: ptr}
		break
	case *BpVoteOperation:
		ptr := op.(*BpVoteOperation)
		res.Op = &Operation_Op5{Op5: ptr}
		break
	case *PostOperation:
		ptr := op.(*PostOperation)
		res.Op = &Operation_Op6{Op6: ptr}
		break
	case *ReplyOperation:
		ptr := op.(*ReplyOperation)
		res.Op = &Operation_Op7{Op7: ptr}
		break
	case *FollowOperation:
		ptr := op.(*FollowOperation)
		res.Op = &Operation_Op8{Op8: ptr}
		break
	case *VoteOperation:
		ptr := op.(*VoteOperation)
		res.Op = &Operation_Op9{Op9: ptr}
		break
	case *TransferToVestingOperation:
		ptr := op.(*TransferToVestingOperation)
		res.Op = &Operation_Op10{Op10: ptr}
		break
	default:
		panic("error op type")
	}
	m.Operations = append(m.Operations, res)
}