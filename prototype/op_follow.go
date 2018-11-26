package prototype

import "github.com/pkg/errors"

func (m *FollowOperation) GetAuthorities(auths *[]Authority) {

}
func (m *FollowOperation) GetRequiredPosting(auths *map[string]bool) {

}

func (m *FollowOperation) GetRequiredOwner(auths *map[string]bool) {

}
func (m *FollowOperation) GetAdmin(*[]AccountAdminPair) {

}
func (m *FollowOperation) IsVirtual() {

}

func (m *FollowOperation) GetRequiredActive(auths *map[string]bool) {
	(*auths)[m.Account.Value] = true
}


func (m *FollowOperation)Validate() error {
	if m == nil {
		return ErrNpe
	}

	if err := m.Account.Validate(); err != nil{
		return errors.WithMessage(err, "Follower error" )
	}

	if err := m.FAccount.Validate(); err != nil{
		return errors.WithMessage(err, "Following error" )
	}

	return nil
}
