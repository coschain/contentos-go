package prototype

import "github.com/pkg/errors"

func (m *BpEnableOperation) GetSigner(auths *map[string]bool) {
	(*auths)[m.Owner.Value] = true

}

func (m *BpEnableOperation) Validate() error {
	if m == nil {
		return ErrNpe
	}

	if err := m.Owner.Validate(); err != nil {
		return errors.WithMessage(err, "Owner error")
	}
	return nil
}

func (m *BpEnableOperation) GetAffectedProps(props *map[string]bool) {
	(*props)["*"] = true
}

func init() {
	registerOperation("bp_enable", (*Operation_Op4)(nil), (*BpEnableOperation)(nil));
}
