package prototype

import (
	"bytes"
	"fmt"
	"github.com/golang/protobuf/jsonpb"
)

func GetBaseOperation(op *Operation) BaseOperation {
	if op != nil && op.Op != nil {
		if o := fromGenericOperation(op); o != nil {
			if base, ok := o.(BaseOperation); ok {
				return base
			}
		}
	}
	return UnknownOperation
}

//Get protoBuffer struct Operation by a interface of detail operation(such as TransferOperation)
func GetPbOperation(op interface{}) *Operation {
	if generic := toGenericOperation(op); generic != nil {
		return generic
	}
	panic(fmt.Sprintf("error op type %v", op))
}

func (op *Operation) MarshalJSON() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := new(jsonpb.Marshaler).Marshal(buf, op); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (op *Operation) UnmarshalJSON(b []byte) error {
	return jsonpb.Unmarshal(bytes.NewReader(b), op)
}
