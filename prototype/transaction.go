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
	case *ReplayOperation:
		ptr := op.(*ReplayOperation)
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