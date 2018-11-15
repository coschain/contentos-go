package prototype

import "github.com/coschain/contentos-go/common/encoding"

func (m *Coin) OpeEncode() ([]byte, error) {
	return encoding.Encode(m.Value)
}

func (m *Coin) NonZero() bool {
	return m.Value != 0
}

func MakeCoin(value uint64) *Coin {
	return &Coin{Value:value}
}