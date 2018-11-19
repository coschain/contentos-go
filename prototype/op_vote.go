package prototype

import "errors"

func (m *VoteOperation) GetAuthorities(auths *[]Authority) {

}
func (m *VoteOperation) GetRequiredPosting(auths *map[string]bool) {

}

func (m *VoteOperation) GetRequiredOwner(auths *map[string]bool) {

}
func (m *VoteOperation) GetAdmin(*[]AccountAdminPair) {

}
func (m *VoteOperation) IsVirtual() {

}

func (m *VoteOperation) GetRequiredActive(auths *map[string]bool) {
	(*auths)[m.Voter.Value] = true
}

func (m *VoteOperation)Validate() error {
	if m == nil {
		return ErrNpe
	}

	if err := m.Voter.Validate(); err != nil{
		return err
	}

	if m.Idx == 0 {
		return errors.New("uuid cant be 0")
	}

	return nil
}