package prototype

import "github.com/pkg/errors"

func (m *BpVoteOperation) GetAuthorities(auths *[]Authority) {

}
func (m *BpVoteOperation) GetRequiredPosting(auths *map[string]bool) {

}

func (m *BpVoteOperation) GetRequiredOwner(auths *map[string]bool) {

}
func (m *BpVoteOperation) GetAdmin(*[]AccountAdminPair) {

}
func (m *BpVoteOperation) IsVirtual() {

}

func (m *BpVoteOperation) GetRequiredActive(auths *map[string]bool) {
	(*auths)[m.Voter.Value] = true
}

func (m *BpVoteOperation) Validate() error {
	if m == nil {
		return ErrNpe
	}

	if err := m.Voter.Validate(); err != nil {
		return errors.WithMessage(err, "Voter error")
	}
	if err := m.Witness.Validate(); err != nil {
		return errors.WithMessage(err, "Witness error")
	}

	return nil
}
