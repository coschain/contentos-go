package prototype

import "github.com/pkg/errors"

func (t *TransferOperation) GetAuthorities(auths *[]Authority) {

}

func (t *TransferOperation) GetAdmin(*[]AccountAdminPair) {

}

func (t *TransferOperation) IsVirtual() {

}

func (t *TransferOperation) GetRequiredOwner(auths *map[string]bool) {
	(*auths)[t.From.Value] = true
}

func (t *TransferOperation) Validate() error {
	if t == nil {
		return ErrNpe
	}
	if err := t.From.Validate(); err != nil {
		return errors.WithMessage(err, "From error")
	}
	if err := t.To.Validate(); err != nil {
		return errors.WithMessage(err, "To error")
	}
	if t.Amount == nil || !t.Amount.NonZero() {
		return errors.New("transfer op must has amount value")
	}
	return nil
}
