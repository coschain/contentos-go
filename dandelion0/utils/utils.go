package utils

import (
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/prototype"
	"time"
)

func GenerateSignedTxAndValidate(ops []interface{}, signer *wallet.PrivAccount) (*prototype.SignedTransaction, error) {
	privKey := &prototype.PrivateKeyType{}

	pk, err := prototype.PrivateKeyFromWIF(signer.PrivKey)
	if err != nil {
		return nil, err
	}
	privKey = pk

	// occupant implement
	tx := &prototype.Transaction{RefBlockNum: 0, RefBlockPrefix: 0, Expiration: &prototype.TimePointSec{UtcSeconds: uint32(time.Now().Unix()) + constants.TrxMaxExpirationTime}}
	for _, op := range ops {
		tx.AddOperation(op)
	}

	signTx := prototype.SignedTransaction{Trx: tx}

	res := signTx.Sign(privKey, prototype.ChainId{Value: 0})
	signTx.Signature = &prototype.SignatureType{Sig: res}


	if err := signTx.Validate(); err != nil {
		return nil, err
	}

	return &signTx, nil
}
