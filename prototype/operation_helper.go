package prototype

import (
	"fmt"
)

func GetBaseOperation(op *Operation) BaseOperation {
	if o := fromGenericOperation(op); o != nil {
		if base, ok := o.(BaseOperation); ok {
			return base
		}
	}
	panic("unknown op type")
}

//Get protoBuffer struct Operation by a interface of detail operation(such as TransferOperation)
func GetPbOperation(op interface{}) *Operation {
	if generic := toGenericOperation(op); generic != nil {
		return generic
	}
	panic(fmt.Sprintf("error op type %v", op))
}
