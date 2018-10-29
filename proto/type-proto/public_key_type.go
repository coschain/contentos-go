package prototype

import (
	"bytes"
	"github.com/coschain/contentos-go/p2p/depend/crypto"
	"crypto/ecdsa"
)

func PublicKeyFromECDSA( key *ecdsa.PublicKey ) *PublicKeyType {
	result := new(PublicKeyType)
	result.Data = crypto.FromECDSAPub(key)
	return result
}

func PublicKeyFromBytes( buffer []byte ) *PublicKeyType {
	result := new(PublicKeyType)
	result.Data = buffer
	return result
}

func (m *PublicKeyType) Equal(other *PublicKeyType) bool {
	return bytes.Equal(m.Data, other.Data)
}

func (m *PublicKeyType) ToECDSA() (*ecdsa.PublicKey, error) {
	return crypto.UnmarshalPubkey( m.Data )
}
