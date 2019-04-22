package utils

import (
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/prototype"
	"time"
)

func GenerateSignedTxAndValidate(ops []interface{}, signers ...*wallet.PrivAccount) (*prototype.SignedTransaction, error) {
	privKeys := []*prototype.PrivateKeyType{}
	for _, acc := range signers {
		privKey, err := prototype.PrivateKeyFromWIF(acc.PrivKey)
		if err != nil {
			return nil, err
		}
		privKeys = append(privKeys, privKey)
	}
	// occupant implement
	tx := &prototype.Transaction{RefBlockNum: 0, RefBlockPrefix: 0, Expiration: &prototype.TimePointSec{UtcSeconds: uint32(time.Now().Unix()) + constants.TRX_MAX_EXPIRATION_TIME}}
	for _, op := range ops {
		tx.AddOperation(op)
	}

	signTx := prototype.SignedTransaction{Trx: tx}
	for _, privkey := range privKeys {
		res := signTx.Sign(privkey, prototype.ChainId{Value: 0})
		signTx.Signatures = append(signTx.Signatures, &prototype.SignatureType{Sig: res})
	}

	if err := signTx.Validate(); err != nil {
		return nil, err
	}

	return &signTx, nil
}
