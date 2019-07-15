package utils

import (
	"fmt"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/prototype"
	"testing"
)

func TestSign(t *testing.T) {
	privKey, _ := prototype.PrivateKeyFromWIF("4DjYx2KAGh1NP3dai7MZTLUBMMhMBPmwouKE8jhVSESywccpVZ")
	//pubkey, _ := privKey.PubKey()
	//keys := prototype.NewAuthorityFromPubKey(pubkey)

	//acop := &prototype.AccountCreateOperation{
	//	Fee:            prototype.NewCoin(1),
	//	Creator:        &prototype.AccountName{Value: "initminer"},
	//	NewAccountName: &prototype.AccountName{Value: "kochiya"},
	//	Owner:          keys,
	//}
	tx := &prototype.Transaction{RefBlockNum: 0, RefBlockPrefix: 0, Expiration: &prototype.TimePointSec{UtcSeconds: 0}}
	value := common.GetChainIdByName("main")
	fmt.Println(value)
	//tx.AddOperation(acop)
	signTx := prototype.SignedTransaction{Trx: tx}
	res := signTx.Sign(privKey, prototype.ChainId{Value: value})
	//signTx.Signatures = append(signTx.Signatures, &prototype.SignatureType{Sig: res})
	fmt.Println(res)
}
