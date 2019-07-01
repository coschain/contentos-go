package prototype

import (
	"github.com/pkg/errors"
)


func (a *AcquireTicketOperation) GetSigner(auths *map[string]bool) {
	(*auths)[a.Account.Value] = true
}


func (a *AcquireTicketOperation) Validate() error {

	if a == nil {
		return ErrNpe
	}

	if err := a.Account.Validate(); err != nil {
		return errors.WithMessage(err, "Account error")
	}

	if a.Count <= 0 {
		return errors.New("Acquire at least 1 ticket")
	}

	return nil
}

func (a *AcquireTicketOperation) GetAffectedProps(props *map[string]bool) {
	(*props)["*"] = true
}

func init() {
	registerOperation("acquire_ticket", (*Operation_Op21)(nil), (*AcquireTicketOperation)(nil));
}
