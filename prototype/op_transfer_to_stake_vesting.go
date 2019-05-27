package prototype

import "github.com/pkg/errors"


func (m *TransferToStakeVestingOperation) GetSigner(auths *map[string]bool) {
	(*auths)[m.From.Value] = true
}

func (m *TransferToStakeVestingOperation) Validate() error {
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

func (m *TransferToStakeVestingOperation) GetAffectedProps(props *map[string]bool) {
	(*props)["*"] = true
}

func init() {
	registerOperation("transfer_to_stake_vesting", (*Operation_Op20)(nil), (*TransferToStakeVestingOperation)(nil));
}
