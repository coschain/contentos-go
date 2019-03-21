package prototype

import (
	"encoding/binary"
	"fmt"
	"github.com/coschain/contentos-go/common"
	"github.com/pkg/errors"
)

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

	for index, op := range m.Operations {
		if err := validateOp(op); err != nil {
			return errors.WithMessage(err, fmt.Sprintf("Operation Error index: %d", index))
		}
	}

	return nil
}

func (m *Transaction) SetReferenceBlock(id *common.BlockID) {
	m.RefBlockNum = uint32(id.BlockNum())
	m.RefBlockPrefix = binary.BigEndian.Uint32(id.Data[8:12])
}

func validateOp(op *Operation) error {
	if op == nil {
		return ErrNpe
	}
	baseOp := GetBaseOperation(op)
	if baseOp == nil {
		return errors.New("unknown op type")
	}
	return baseOp.Validate()
}

func (m *Transaction) AddOperation(op interface{}) {
	res := GetPbOperation(op)
	m.Operations = append(m.Operations, res)
}

func (tx *Transaction) GetAffectedProps(props *map[string]bool) {
	p := make(map[string]bool)
	for _, op := range tx.GetOperations() {
		GetBaseOperation(op).GetAffectedProps(&p)
	}
	if p["*"] {
		(*props)["*"] = true
	} else {
		for k, v := range p {
			(*props)[k] = v
		}
	}
}
