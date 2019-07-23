package prototype

import "github.com/coschain/contentos-go/common/encoding/kope"

func (m *StakeRecord) OpeEncode() ([]byte, error) {
	return kope.Encode(m.From.Value, m.To.Value)
}

func (m *StakeRecordReverse) OpeEncode() ([]byte, error) {
	return kope.Encode(m.To.Value, m.From.Value)
}