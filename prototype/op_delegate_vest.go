package prototype

import (
	"github.com/coschain/contentos-go/common/constants"
	"github.com/pkg/errors"
)


func (m *DelegateVestOperation) GetSigner(auths *map[string]bool) {
	(*auths)[m.GetFrom().GetValue()] = true
}

func (m *DelegateVestOperation) Validate() error {
	if m == nil {
		return ErrNpe
	}
	if err := m.GetFrom().Validate(); err != nil {
		return errors.WithMessage(err, "from account error")
	}
	if err := m.GetTo().Validate(); err != nil {
		return errors.WithMessage(err, "to account error")
	}
	if m.GetAmount().GetValue() < constants.MinVestDelegationAmount {
		return errors.New("amount too small")
	}
	if m.GetExpiration() < constants.MinVestDelegationInBlocks {
		return errors.New("expiration too short")
	}
	if m.GetExpiration() > constants.MaxVestDelegationInBlocks {
		return errors.New("expiration too long")
	}
	return nil
}

func (m *DelegateVestOperation) GetAffectedProps(props *map[string]bool) {
	(*props)["*"] = true
}

func init() {
	registerOperation("delegate_vest", (*Operation_Op23)(nil), (*DelegateVestOperation)(nil))
}
