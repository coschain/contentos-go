package prototype

import "github.com/pkg/errors"

func (t *ConvertVestOperation) GetSigner(auths *map[string]bool) {
	(*auths)[t.From.Value] = true
}

func (t *ConvertVestOperation) Validate() error {
	if t == nil {
		return ErrNpe
	}
	if err := t.From.Validate(); err != nil {
		return errors.WithMessage(err, "From error")
	}
	if t.Amount == nil || t.Amount.Value < 1e6 {
		return errors.New("Amount field is required by convert vest operation and the value should greater than 1000000 (1 cos)")
	}
	return nil
}

func (m *ConvertVestOperation) GetAffectedProps(props *map[string]bool) {
	(*props)["*"] = true
}

func init() {
	registerOperation("convert_vest", (*Operation_Op16)(nil), (*ConvertVestOperation)(nil));
}
