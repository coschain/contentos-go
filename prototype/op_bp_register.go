package prototype

import "github.com/pkg/errors"


func (m *BpRegisterOperation) GetSigner(auths *map[string]bool) {
	(*auths)[m.Owner.Value] = true
}

func (m *BpRegisterOperation) Validate() error {
	if m == nil {
		return ErrNpe
	}

	if err := m.Owner.Validate(); err != nil {
		return errors.WithMessage(err, "Owner error")
	}
	if err := m.BlockSigningKey.Validate(); err != nil {
		return errors.WithMessage(err, "BlockSigningKey error")
	}
	if m.Props == nil {
		return ErrNpe
	}

	// TODO chain property valid check
	return nil
}
