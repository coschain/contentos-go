package prototype

import "github.com/coschain/contentos-go/common/encoding/kope"

func (m *ContractId) OpeEncode() ([]byte, error) {
	return kope.Encode(m.Owner, m.Cname)
}

func (m *ContractDataId) OpeEncode() ([]byte, error) {
	return kope.Encode(m.Owner, m.Cname, m.Pos)
}
