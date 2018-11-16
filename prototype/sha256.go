package prototype

import "github.com/coschain/contentos-go/common/encoding"

func (m *Sha256) OpeEncode() ([]byte, error) {
	return encoding.Encode(m.Hash)
}


func (m *Sha256) Validate () error {
	if m == nil{
		return ErrNpe
	}
	if len(m.Hash) != 32{
		return ErrHashLength
	}
	return nil
}