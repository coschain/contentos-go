package prototype

import (
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/encoding/kope"
)

func (m *Sha256) OpeEncode() ([]byte, error) {
	return kope.Encode(m.Hash)
}

func (m *Sha256) FromBlockID(id common.BlockID) {
	m.Hash = make([]byte, 32)
	copy(m.Hash, id.Data[:])
}

func (m *Sha256) Validate() error {
	if m == nil {
		return ErrNpe
	}
	if len(m.Hash) != 32 {
		return ErrHashLength
	}
	return nil
}
