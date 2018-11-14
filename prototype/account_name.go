package prototype

import "github.com/coschain/contentos-go/common/encoding"

func (m *AccountName) Empty() bool {
	return m.Value == ""
}

func (m *AccountName) OpeEncode() ([]byte, error) {
	return encoding.Encode(m.Value)
}

func MakeAccountName(value string) *AccountName {
	return &AccountName{Value: value}
}