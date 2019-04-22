package prototype

import (
	"github.com/coschain/contentos-go/common/encoding/kope"
	"github.com/pkg/errors"
)

func (m *AccountName) Empty() bool {
	return m.Value == ""
}

func (m *AccountName) OpeEncode() ([]byte, error) {
	return kope.Encode(m.Value)
}

func isValidNameChar(c byte) bool {
	if c >= '0' && c <= '9' {
		return true
	} else if c >= 'a' && c <= 'z' {
		return true
	} else if c >= 'A' && c <= 'Z' {
		return true
	} else {
		return false
	}
}

func (m *AccountName) Validate() error {
	if m == nil {
		return errors.New("npe")
	}

	if len(m.Value) < 6 || len(m.Value) > 16 {
		return errors.New("name length invalid: " + m.Value)
	}

	buf := []byte(m.Value)

	for _, val := range buf {
		if !isValidNameChar(val) {
			return errors.New("name contains invalid char: " + string(val))
		}
	}
	return nil
}

func NewAccountName(value string) *AccountName {
	return &AccountName{Value: value}
}
