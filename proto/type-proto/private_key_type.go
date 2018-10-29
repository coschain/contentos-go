package prototype

import (
	"bytes"
	"github.com/coschain/contentos-go/p2p/depend/crypto"
	"crypto/ecdsa"
)

func PrivateKeyFromECDSA( key *ecdsa.PrivateKey ) *PrivateKeyType {
	result := new(PrivateKeyType)
	result.Data = crypto.FromECDSA(key)
	return result
}

func GenerateNewKey() (*PrivateKeyType, error) {
	sigRawKey, err := crypto.GenerateKey()

	if err != nil{
		return nil, err
	}

	return PrivateKeyFromECDSA(sigRawKey), nil
}

func PrivateKeyFromBytes( buffer []byte ) *PrivateKeyType {
	result := new(PrivateKeyType)
	result.Data = buffer
	return result
}

func (m *PrivateKeyType) Equal(other *PrivateKeyType) bool {
	return bytes.Equal(m.Data, other.Data)
}

func (m *PrivateKeyType) ToECDSA() (*ecdsa.PrivateKey, error) {
	return crypto.ToECDSA(m.Data)
}

func (m *PrivateKeyType) PubKey() (*PublicKeyType, error)  {
	ecKey, err := m.ToECDSA()

	if err != nil{
		return nil, err
	}
	return PublicKeyFromECDSA( &ecKey.PublicKey ), nil
}
