package prototype

import "github.com/pkg/errors"

func (m *BpRegisterOperation) GetAuthorities(auths *[]Authority) {

}

func (m *BpRegisterOperation) GetRequiredOwner(auths *map[string]bool) {
	(*auths)[m.Owner.Value] = true
}
func (m *BpRegisterOperation) GetAdmin(*[]AccountAdminPair) {

}
func (m *BpRegisterOperation) IsVirtual() {

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
