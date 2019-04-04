package prototype

import "github.com/pkg/errors"

func (t *ConvertVestingOperation) GetSigner(auths *map[string]bool) {
	(*auths)[t.From.Value] = true
}

func (t *ConvertVestingOperation) Validate() error {
	if t == nil {
		return ErrNpe
	}
	if err := t.From.Validate(); err != nil {
		return errors.WithMessage(err, "From error")
	}
	if t.Amount == nil || t.Amount.Value < 1e6 {
		return errors.New("convert vesting op must has amount value and it should greater than 1000000 (1 cos)")
	}
	return nil
}

func (m *ConvertVestingOperation) GetAffectedProps(props *map[string]bool) {
	(*props)["*"] = true
}
