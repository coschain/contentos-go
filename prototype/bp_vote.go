package prototype

import "github.com/coschain/contentos-go/common/encoding/kope"

func (m *BpBlockProducerId) OpeEncode() ([]byte, error) {
	return kope.Encode(m.BlockProducer.Value, m.Voter.Value)
}
