package prototype

import "github.com/pkg/errors"


func (m *TransferToVestOperation) GetSigner(auths *map[string]bool) {
	(*auths)[m.From.Value] = true
}

func (m *TransferToVestOperation) Validate() error {
	if m == nil {
		return ErrNpe
	}

	if err := m.From.Validate(); err != nil {
		return errors.WithMessage(err, "From error")
	}

	if err := m.To.Validate(); err != nil {
		return errors.WithMessage(err, "To error")
	}

	if m.Amount == nil || m.Amount.Value == 0 {
		return errors.New("amount cant be 0")
	}

	return nil
}

func (m *TransferToVestOperation) GetAffectedProps(props *map[string]bool) {
	(*props)["*"] = true
}

func init() {
	registerOperation("transfer_to_vest", (*Operation_Op10)(nil), (*TransferToVestOperation)(nil));
}
