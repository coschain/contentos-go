package prototype

import "github.com/coschain/contentos-go/common/encoding"

func (m *AccountName) Empty() bool {
	return m.Value == ""
}

func (m *AccountName) OpeEncode() ([]byte, error) {
	return encoding.Encode(m.Value)
}

func isValidNameChar( c byte ) bool {
	if c >='0' && c <= '9'{
		return true
	} else if c >='a' && c <= 'z'{
		return true
	} else if c >='A' && c <= 'Z'{
		return true
	} else {
		return false
	}
}

func (m *AccountName) Validate() bool {
	if m == nil {
		return false
	}

	if len(m.Value) < 6 || len(m.Value) > 16 {
		return false
	}

	buf := []byte(m.Value)

	for _, val := range buf {
		if !isValidNameChar(val){
			return false
		}
	}
	return true
}

func MakeAccountName(value string) *AccountName {
	return &AccountName{Value: value}
}