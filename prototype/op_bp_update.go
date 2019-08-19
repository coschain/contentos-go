package prototype

import "github.com/pkg/errors"

func (m *BpUpdateOperation) GetSigner(auths *map[string]bool) {
	(*auths)[m.Owner.Value] = true
}

func (m *BpUpdateOperation) Validate() error {
	if m == nil {
		return ErrNpe
	}
	if err := m.Owner.Validate(); err != nil {
		return errors.WithMessage(err, "Owner error")
	}
	// TODO: chain_properties check
	return nil
}

func (m *BpUpdateOperation) GetAffectedProps(props *map[string]bool) {
	(*props)["*"] = true
}

func init() {
	registerOperation("bp_update", (*Operation_Op19)(nil), (*BpUpdateOperation)(nil));
}
