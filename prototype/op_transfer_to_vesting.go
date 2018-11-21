package prototype

import "github.com/pkg/errors"

func (m *TransferToVestingOperation) GetAuthorities(auths *[]Authority) {

}
func (m *TransferToVestingOperation) GetRequiredPosting(auths *map[string]bool) {

}

func (m *TransferToVestingOperation) GetAdmin(*[]AccountAdminPair) {

}
func (m *TransferToVestingOperation) IsVirtual() {

}

func (m *TransferToVestingOperation) GetRequiredOwner(auths *map[string]bool) {
	(*auths)[m.From.Value] = true
}

func (m *TransferToVestingOperation) GetRequiredActive(auths *map[string]bool) {
	(*auths)[m.From.Value] = true
}

func (m *TransferToVestingOperation) Validate() error {
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
