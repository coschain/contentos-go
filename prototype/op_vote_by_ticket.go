package prototype

import (
	"github.com/pkg/errors"
)


func (a *VoteByTicketOperation) GetSigner(auths *map[string]bool) {
	(*auths)[a.Account.Value] = true
}


func (a *VoteByTicketOperation) Validate() error {
	if a == nil {
		return ErrNpe
	}

	if err := a.Account.Validate(); err != nil {
		return errors.WithMessage(err, "account error")
	}

	if a.Count <= 0 {
		return errors.New("vote at least 1 ticket")
	}

	return nil
}

func (a *VoteByTicketOperation) GetAffectedProps(props *map[string]bool) {
	(*props)["*"] = true
}

func init() {
	registerOperation("vote_by_ticket", (*Operation_Op22)(nil), (*VoteByTicketOperation)(nil));
}
