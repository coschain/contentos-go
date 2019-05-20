package prototype

import (
	"bytes"
	"encoding/hex"
	"fmt"
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


func (m *Sha256) Equal( p* Sha256) bool {
	return bytes.Equal(m.Hash, p.Hash)
}


func (m *Sha256) ToString() string {
	return hex.EncodeToString(m.Hash)
}

func (m *Sha256) MarshalJSON() ([]byte, error) {
	val := fmt.Sprintf("\"%s\"", m.ToString())
	return []byte(val), nil
}


func (m *Sha256) UnmarshalJSON(input []byte) error {

	strBuffer, err := stripJsonQuota(input)
	if err != nil {
		return err
	}

	res, err := hex.DecodeString( string(strBuffer) )
	if err != nil {
		return err
	}
	m.Hash = res
	return nil
}