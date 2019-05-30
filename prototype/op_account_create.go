package prototype

import (
	"github.com/pkg/errors"
)


func (a *AccountCreateOperation) GetSigner(auths *map[string]bool) {
	(*auths)[a.Creator.Value] = true
}


func (a *AccountCreateOperation) Validate() error {

	if a == nil {
		return ErrNpe
	}

	if err := a.Creator.Validate(); err != nil {
		return errors.WithMessage(err, "Creator error")

	}

	if err := a.NewAccountName.Validate(); err != nil {
		return errors.WithMessage(err, "NewAccountName error")
	}

	if a.Owner == nil {
		return errors.New("Posting Key cant be empty")
	}
	if err := a.Owner.Validate(); err != nil {
		return errors.WithMessage(err, "Owner error")
	}

	if a.Fee == nil || a.Fee.Value == 0 {
		return errors.New("Account Create must set Fee")
	}

	return nil
}

func (a *AccountCreateOperation) GetAffectedProps(props *map[string]bool) {
	(*props)["*"] = true
}

func init() {
	registerOperation("account_create", (*Operation_Op1)(nil), (*AccountCreateOperation)(nil));
}
