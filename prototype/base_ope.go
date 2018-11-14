package prototype

import "github.com/coschain/contentos-go/common/encoding"

func (m *AccountName) OpeEncode() ([]byte, error) {
	return encoding.Encode(m.Value)
}

func (m *PublicKeyType) OpeEncode() ([]byte, error) {
	return encoding.Encode(m.Data)
}

func (m *TimePointSec) OpeEncode() ([]byte, error) {
	return encoding.Encode(m.UtcSeconds)
}

func (m *Safe64) OpeEncode() ([]byte, error) {
	return encoding.Encode(m.Value)
}

func (m *Coin) OpeEncode() ([]byte, error) {
	return m.Amount.OpeEncode()
}

func (m *Vest) OpeEncode() ([]byte, error) {
	return m.Amount.OpeEncode()
}

func MakeSafe64(value int64) *Safe64 {
	return &Safe64{Value: value}
}

func MakeTimeSecondPoint(value uint32) *TimePointSec {
	return &TimePointSec{UtcSeconds: value}
}

func MakeAccountName(value string) *AccountName {
	return &AccountName{Value: value}
}

func MakeCoin(value int64) *Coin {
	return &Coin{Amount: MakeSafe64(value)}
}

func MakeVest(value int64) *Vest {
	return &Vest{Amount: MakeSafe64(value)}
}

func MakePublicKeyType(buf []byte) *PublicKeyType {
	return PublicKeyFromBytes(buf)
}
