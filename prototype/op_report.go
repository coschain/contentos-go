package prototype

import "github.com/pkg/errors"


func (m *ReportOperation) GetSigner(auths *map[string]bool) {
	(*auths)[m.Reporter.Value] = true
}

func (m *ReportOperation) Validate() error {
	if m == nil {
		return ErrNpe
	}

	if err := m.Reporter.Validate(); err != nil {
		return errors.WithMessage(err, "Owner error")
	}

	if m.Reported == 0 {
		return errors.New("uuid cant be 0")
	}

	if len(m.ReportTag) > 5 {
		return errors.New("too many tags")
	}

	return nil
}

func (m *ReportOperation) GetAffectedProps(props *map[string]bool) {
	(*props)["*"] = true
}

//func init() {
//	registerOperation("report", (*Operation_Op15)(nil), (*ReportOperation)(nil))
//}
