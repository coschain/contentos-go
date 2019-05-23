package prototype

import "github.com/pkg/errors"

func (m *FollowOperation) GetSigner(auths *map[string]bool) {
	(*auths)[m.Account.Value] = true
}

func (m *FollowOperation) Validate() error {
	if m == nil {
		return ErrNpe
	}

	if err := m.Account.Validate(); err != nil {
		return errors.WithMessage(err, "Follower error")
	}

	if err := m.FAccount.Validate(); err != nil {
		return errors.WithMessage(err, "Following error")
	}

	return nil
}

func (m *FollowOperation) GetAffectedProps(props *map[string]bool) {
	(*props)["*"] = true
}

func init() {
	registerOperation("follow", (*Operation_Op8)(nil), (*FollowOperation)(nil));
}
