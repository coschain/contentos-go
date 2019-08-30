package prototype

import (
	"github.com/coschain/contentos-go/common/constants"
	"github.com/pkg/errors"
)

func (m *ReplyOperation) GetSigner(auths *map[string]bool) {
	(*auths)[m.Owner.Value] = true
}

func (m *ReplyOperation) Validate() error {
	if m == nil {
		return ErrNpe
	}

	if err := m.Owner.Validate(); err != nil {
		return errors.WithMessage(err, "Follower error")
	}

	if m.Uuid == constants.PostInvalidId {
		return errors.New("uuid cant be 0")
	}

	if m.ParentUuid == constants.PostInvalidId {
		return errors.New("parent uuid cant be 0")
	}
	if len(m.Content) == 0 {
		return errors.New("content cant be null")
	}

	return nil
}

func (m *ReplyOperation) GetAffectedProps(props *map[string]bool) {
	(*props)["*"] = true
}

func init() {
	registerOperation("reply", (*Operation_Op7)(nil), (*ReplyOperation)(nil));
}
