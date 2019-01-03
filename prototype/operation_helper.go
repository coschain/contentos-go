package prototype

import (
	"fmt"
)

//Get a BaseOperation by a Operation struct
func GetBaseOperation(op *Operation) BaseOperation {
	switch t := op.Op.(type) {
	case *Operation_Op1:
		return BaseOperation(t.Op1)
	case *Operation_Op2:
		return BaseOperation(t.Op2)
	case *Operation_Op3:
		return BaseOperation(t.Op3)
	case *Operation_Op4:
		return BaseOperation(t.Op4)
	case *Operation_Op5:
		return BaseOperation(t.Op5)
	case *Operation_Op6:
		return BaseOperation(t.Op6)
	case *Operation_Op7:
		return BaseOperation(t.Op7)
	case *Operation_Op8:
		return BaseOperation(t.Op8)
	case *Operation_Op9:
		return BaseOperation(t.Op9)
	case *Operation_Op10:
		return BaseOperation(t.Op10)
	// TODO @zengli
	case *Operation_Op11:
		return BaseOperation(t.Op11)
	case *Operation_Op12:
		return BaseOperation(t.Op12)
	case *Operation_Op13:
		return BaseOperation(t.Op13)
	case *Operation_Op14:
		return BaseOperation(t.Op14)
	case *Operation_Op15:
		return BaseOperation(t.Op15)
	default:
		panic("unknown op type")
	}
	return nil
}

//Get protoBuffer struct Operation by a interface of detail operation(such as TransferOperation)
func GetPbOperation(op interface{}) *Operation {

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
	case *ClaimOperation:
		ptr := op.(*ClaimOperation)
		res.Op = &Operation_Op11{Op11: ptr}
		break
	case *ClaimAllOperation:
		ptr := op.(*ClaimAllOperation)
		res.Op = &Operation_Op12{Op12: ptr}
		break
	case *ContractDeployOperation:
		ptr := op.(*ContractDeployOperation)
		res.Op = &Operation_Op13{Op13: ptr}
		break
	case *ContractApplyOperation:
		ptr := op.(*ContractApplyOperation)
		res.Op = &Operation_Op14{Op14: ptr}
		break
	case *ContractEstimateApplyOperation:
		ptr := op.(*ContractEstimateApplyOperation)
		res.Op = &Operation_Op15{Op15: ptr}
	default:
		panic(fmt.Sprintf("error op type %v", op))
	}
	return res
}
