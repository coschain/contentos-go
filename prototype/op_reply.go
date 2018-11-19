package prototype

import "github.com/pkg/errors"

func (m *ReplyOperation) GetAuthorities(auths *[]Authority) {

}
func (m *ReplyOperation) GetRequiredPosting(auths *map[string]bool) {

}

func (m *ReplyOperation) GetRequiredOwner(auths *map[string]bool) {

}
func (m *ReplyOperation) GetAdmin(*[]AccountAdminPair) {

}
func (m *ReplyOperation) IsVirtual() {

}

func (m *ReplyOperation) GetRequiredActive(auths *map[string]bool) {
	(*auths)[m.Owner.Value] = true
}


func (m *ReplyOperation)Validate() error {
	if m == nil {
		return ErrNpe
	}

	if err := m.Owner.Validate(); err != nil{
		return errors.WithMessage(err, "Follower error" )
	}

	if m.Uuid == 0 {
		return errors.New("uuid cant be 0")
	}

	if m.ParentUuid == 0 {
		return errors.New("parent uuid cant be null")
	}
	if len(m.Content) == 0 {
		return errors.New("content cant be null")
	}

	return nil
}
