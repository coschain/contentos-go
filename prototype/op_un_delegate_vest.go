package prototype

import (
	"github.com/pkg/errors"
)


func (m *UnDelegateVestOperation) GetSigner(auths *map[string]bool) {
	(*auths)[m.GetAccount().GetValue()] = true
}

func (m *UnDelegateVestOperation) Validate() error {
	if m == nil {
		return ErrNpe
	}
	if m.GetOrderId() == 0 {
		return errors.New("invalid order id")
	}
	return nil
}

func (m *UnDelegateVestOperation) GetAffectedProps(props *map[string]bool) {
	(*props)["*"] = true
}

func init() {
	registerOperation("un_delegate_vest", (*Operation_Op24)(nil), (*UnDelegateVestOperation)(nil))
}
