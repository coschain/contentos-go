package prototype

import "github.com/coschain/contentos-go/common/encoding"

func (m *Safe64) Min() *Safe64 {
	return &Safe64{Value: 0}
}


func (m *Safe64) OpeEncode() ([]byte, error) {
	return encoding.Encode(m.Value)
}


func MakeSafe64(value int64) *Safe64 {
	return &Safe64{Value: value}
}