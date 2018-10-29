package prototype

import (
	"github.com/gogo/protobuf/proto"
	"crypto/sha256"
	cmn "github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/p2p/depend/crypto/secp256k1"
	"errors"
)

func (p* SignedTransaction) ExportPubKey( cid ChainId) ([]byte, error) {
	buf, err := p.GetTrxHash(cid)

	if err != nil{
		return nil, errors.New("sha256 error")
	}

	return secp256k1.RecoverPubkey( buf, p.Signatures[0].Sig )
}

func (p *SignedTransaction) Validate( cid ChainId ) bool {
	return true
}


func (p *SignedTransaction) VerifySig(pubKey []byte, cid ChainId ) bool {

	buf, err := p.GetTrxHash(cid)

	if err != nil{
		return false
	}

	res := secp256k1.VerifySignature( pubKey, buf, p.Signatures[0].Sig[0:64] )

	return res
}

func (p *SignedTransaction) GetTrxHash(cid ChainId ) ([]byte, error)  {
	buf, err := proto.Marshal(p.Trx)

	if err != nil{
		return nil, err
	}

	h := sha256.New()

	cidBuf := cmn.Int2Bytes( cid.Value )
	h.Reset()
	h.Write( cidBuf )
	h.Write( buf )
	bs := h.Sum(nil)

	if bs == nil{
		return nil, errors.New("sha256 error")
	}

	return bs,nil
}


func (p *SignedTransaction) Sign(secKey []byte, cid ChainId ) []byte {

	buf, err := p.GetTrxHash(cid)

	if err != nil{
		return nil
	}

	res, err := secp256k1.Sign(buf,secKey)

	if err != nil{
		return nil
	}

	return res
}