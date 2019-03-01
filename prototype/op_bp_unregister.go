package prototype

import "github.com/pkg/errors"

func (m *BpUnregisterOperation) GetSigner(auths *map[string]bool) {
	(*auths)[m.Owner.Value] = true

}

func (m *BpUnregisterOperation) Validate() error {
	if m == nil {
		return ErrNpe
	}

	if err := m.Owner.Validate(); err != nil {
		return errors.WithMessage(err, "Owner error")
	}
	return nil
}
