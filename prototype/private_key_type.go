package prototype

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/common/crypto"
	"github.com/coschain/contentos-go/common/crypto/secp256k1"
	"github.com/itchyny/base58-go"
	"math/big"
)

func PrivateKeyFromECDSA(key *ecdsa.PrivateKey) *PrivateKeyType {
	result := new(PrivateKeyType)
	result.Data = crypto.FromECDSA(key)
	return result
}

func PrivateKeyFromWIF(encoded string) (*PrivateKeyType, error) {
	if encoded == "" {
		return nil, errors.New("invalid address 1")
	}
	decoded, err := base58.BitcoinEncoding.Decode([]byte(encoded))
	if err != nil {
		return nil, err
	}

	x, ok := new(big.Int).SetString(string(decoded), 10)
	if !ok {
		return nil, errors.New("invalid address 2")
	}

	buf := x.Bytes()
	if len(buf) <= 5 || buf[0] != 1 {
		return nil, errors.New("invalid address 3")
	}
	buf = buf[1:]

	temp := sha256.Sum256(buf[:len(buf)-4])
	temps := sha256.Sum256(temp[:])

	if !bytes.Equal(temps[0:4], buf[len(buf)-4:]) {
		return nil, errors.New("invalid address 4")
	}

	return PrivateKeyFromBytes(buf[:len(buf)-4]), nil
}


// DANGER !!!!!!!!!!!!!!!!!
// this function only for test case
// If used improperly, the private key will be exhausted
func GenerateNewKeyFromBytes(buff []byte) (*PrivateKeyType, error) {

	cBuff1 := sha256.Sum256(buff)

	cBuff := make([]byte, 0)
	cBuff = append(cBuff, cBuff1[:]...)
	cBuff = append(cBuff, cBuff1[:]...)

	sigRawKey, err := crypto.GenerateKeyFromBytes(cBuff)

	if err != nil {
		return nil, err
	}

	return PrivateKeyFromECDSA(sigRawKey), nil
}

func GenerateNewKey() (*PrivateKeyType, error) {
	sigRawKey, err := crypto.GenerateKey()

	if err != nil {
		return nil, err
	}

	return PrivateKeyFromECDSA(sigRawKey), nil
}

func PrivateKeyFromBytes(buffer []byte) *PrivateKeyType {
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

func (m *PrivateKeyType) PubKey() (*PublicKeyType, error) {

	sigRaw, err := crypto.ToECDSA(m.Data)
	if err != nil {
		return nil, err
	}
	buf := secp256k1.CompressPubkey(sigRaw.PublicKey.X, sigRaw.PublicKey.Y)
	return PublicKeyFromBytes(buf), nil
}

func (m *PrivateKeyType) ToWIF() string {
	return m.ToBase58()
}

// ToBase58 returns base58 encoded address string
func (m *PrivateKeyType) ToBase58() string {
	data := m.Data
	temp := sha256.Sum256(data)
	temps := sha256.Sum256(temp[:])
	data = append(data, temps[0:4]...)

	// this avoids any data with leading 0x00 bytes,
	// because leading 0x00 bytes can't survive the base58->private_key decoding.
	xdata := bytes.Join([][]byte{ {1}, data }, nil)

	bi := new(big.Int).SetBytes(xdata).String()
	encoded, _ := base58.BitcoinEncoding.Encode([]byte(bi))
	return string(encoded)
}

func (m *PrivateKeyType) Validate() error {
	if m == nil {
		return ErrNpe
	}
	if len(m.Data) != 32 {
		return ErrKeyLength
	}
	return nil
}

func (m *PrivateKeyType) MarshalJSON() ([]byte, error) {
	val := fmt.Sprintf("\"%s\"", m.ToWIF())
	return []byte(val), nil
}

func (m *PrivateKeyType) UnmarshalJSON(input []byte) error {

	if len(input) < 2 {
		return errors.New("private key length error")
	}
	if input[0] != '"' {
		return errors.New("private key error")
	}
	if input[len(input)-1] != '"' {
		return errors.New("private key error")
	}

	res, err := PrivateKeyFromWIF(string(input[1 : len(input)-1]))
	if err != nil {
		return err
	}
	m.Data = res.Data
	return nil
}
