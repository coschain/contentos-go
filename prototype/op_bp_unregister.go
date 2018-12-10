package prototype

import "github.com/pkg/errors"

func (m *BpUnregisterOperation) GetAuthorities(auths *[]Authority) {

}

func (m *BpUnregisterOperation) GetRequiredOwner(auths *map[string]bool) {
	(*auths)[m.Owner.Value] = true

}
func (m *BpUnregisterOperation) GetAdmin(*[]AccountAdminPair) {

}
func (m *BpUnregisterOperation) IsVirtual() {

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
