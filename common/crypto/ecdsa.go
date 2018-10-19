package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"math/big"
)

// SigPrivKey ecdsa private key
type SigPrivKey struct {
	p *ecdsa.PrivateKey
}

// ToString convert the private key to string
func (spk *SigPrivKey) ToString() (string, error) {
	x509Encoded, err := x509.MarshalECPrivateKey(spk.p)
	if err != nil {
		return "", err
	}
	pemEncoded := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: x509Encoded})
	return string(pemEncoded), nil
}

// Public returns public key correspond to private key
func (spk *SigPrivKey) Public() *SigPubKey {
	pubKey := &spk.p.PublicKey
	return &SigPubKey{
		p: pubKey,
	}
}

// Sign signs digest
func (spk *SigPrivKey) Sign(digest []byte) ([]byte, error) {
	r, s, err := ecdsa.Sign(rand.Reader, spk.p, digest[:])
	if err != nil {
		return nil, err
	}

	return asn1.Marshal(ecdsaSignature{r, s})
}

type ecdsaSignature struct {
	R, S *big.Int
}

// SigPubKey ecdsa public key
type SigPubKey struct {
	p *ecdsa.PublicKey
}

// ToString convert the public key to string
func (spk *SigPubKey) ToString() (string, error) {
	x509EncodedPub, err := x509.MarshalPKIXPublicKey(spk.p)
	if err != nil {
		return "", err
	}
	pemEncodedPub := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: x509EncodedPub})
	return string(pemEncodedPub), nil
}

// Verify verifies the sig against the digest
// NOTE that digest has to be a sha256
func (spk *SigPubKey) Verify(digest, sig []byte) bool {
	var ecdsaSig ecdsaSignature
	asn1.Unmarshal(sig, &ecdsaSig)
	return ecdsa.Verify(spk.p, digest, ecdsaSig.R, ecdsaSig.S)
}

// GenerateKey generates a private key
func GenerateKey() (*SigPrivKey, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	return &SigPrivKey{
		p: privateKey,
	}, nil
}

// ConstructKeyFromString create a private key from string
func ConstructKeyFromString(keyStr string) (*SigPrivKey, error) {
	block, _ := pem.Decode([]byte(keyStr))
	x509Encoded := block.Bytes
	privateKey, err := x509.ParseECPrivateKey(x509Encoded)
	if err != nil {
		return nil, err
	}

	return &SigPrivKey{
		p: privateKey,
	}, nil
}

// ConstructPubKeyFromString create a public key from string
func ConstructPubKeyFromString(pubKeyStr string) (*SigPubKey, error) {
	blockPub, _ := pem.Decode([]byte(pubKeyStr))
	x509EncodedPub := blockPub.Bytes
	genericPublicKey, err := x509.ParsePKIXPublicKey(x509EncodedPub)
	if err != nil {
		return nil, err
	}
	publicKey := genericPublicKey.(*ecdsa.PublicKey)

	return &SigPubKey{
		p: publicKey,
	}, nil
}
