package prototype

import "github.com/pkg/errors"


func (m *PostOperation) GetSigner(auths *map[string]bool) {
	(*auths)[m.Owner.Value] = true
}

func (m *PostOperation) Validate() error {
	if m == nil {
		return ErrNpe
	}

	if err := m.Owner.Validate(); err != nil {
		return errors.WithMessage(err, "Owner error")
	}

	if m.Uuid == 0 {
		return errors.New("uuid cant be 0")
	}

	if len(m.Title) == 0 {
		return errors.New("title cant be null")
	}
	if len(m.Content) == 0 {
		return errors.New("content cant be null")
	}
	if len(m.Tags) == 0 {
		return errors.New("tags cant be null")
	}

	for _, val := range m.Tags {
		if len(val) == 0 {
			return errors.New("tag length cant be null")
		}
	}

	return nil
}

func (m *PostOperation) GetAffectedProps(props *map[string]bool) {

}
