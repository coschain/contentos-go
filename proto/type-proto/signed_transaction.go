package prototype

import (
	"crypto/sha256"
	"errors"
	cmn "github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/p2p/depend/crypto"
	"github.com/coschain/contentos-go/p2p/depend/crypto/secp256k1"
	"github.com/gogo/protobuf/proto"
)

func (p *SignedTransaction) ExportPubKeys(cid ChainId) ([]*PublicKeyType, error) {
	buf, err := p.GetTrxHash(cid)

	if err != nil {
		return nil, errors.New("sha256 error")
	}

	if len(p.Signatures) == 0 {
		return nil, errors.New("no signatures")
	}

	result := make([]*PublicKeyType, len(p.Signatures))

	for index, sig := range p.Signatures {
		buffer, err := secp256k1.RecoverPubkey(buf, sig.Sig)

		if err != nil {
			return nil, errors.New("recover error")
		}

		ecPubKey, err := crypto.UnmarshalPubkey(buffer)
		if err != nil {
			return nil, errors.New("recover error")
		}

		result[index] = PublicKeyFromBytes(secp256k1.CompressPubkey(ecPubKey.X, ecPubKey.Y))
	}

	return result, nil
}

func (p *SignedTransaction) Validate() {
}

func (p *SignedTransaction) VerifySig(pubKey *PublicKeyType, cid ChainId) bool {

	buf, err := p.GetTrxHash(cid)

	if err != nil {
		return false
	}

	for _, sig := range p.Signatures {
		if secp256k1.VerifySignature(pubKey.Data, buf, sig.Sig[0:64]) {
			return true
		}
	}

	return false
}

func (p *SignedTransaction) GetTrxHash(cid ChainId) ([]byte, error) {
	buf, err := proto.Marshal(p.Trx)

	if err != nil {
		return nil, err
	}

	h := sha256.New()

	cidBuf := cmn.Int2Bytes(cid.Value)
	h.Reset()
	h.Write(cidBuf)
	h.Write(buf)
	bs := h.Sum(nil)

	if bs == nil {
		return nil, errors.New("sha256 error")
	}

	return bs, nil
}

func (p *SignedTransaction) Sign(secKey *PrivateKeyType, cid ChainId) []byte {

	buf, err := p.GetTrxHash(cid)

	if err != nil {
		return nil
	}

	res, err := secp256k1.Sign(buf, secKey.Data)

	if err != nil {
		return nil
	}

	return res
}

func (p *SignedTransaction) Id() (*Sha256, error) {
	buf, err := proto.Marshal(p.Trx)
	if err != nil {
		return nil, err
	}
	h := sha256.New()
	h.Reset()
	h.Write(buf)
	bs := h.Sum(nil)
	if bs == nil {
		return nil, errors.New("sha256 error")
	}
	id := &Sha256{Hash: bs}
	return id, nil
}

func (p *SignedTransaction) verifyAuthority(cid ChainId) {
	/*	pubkeys, err := p.ExportPubKeys(cid)
		if err != nil {
			panic(err)
		}*/
}
