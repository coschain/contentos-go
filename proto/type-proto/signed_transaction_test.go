package prototype

import "testing"
import (
	"fmt"
	"github.com/coschain/contentos-go/p2p/depend/crypto"
)

func makeOp() *TransferOperation {
	top := &TransferOperation{
		From:   &AccountName{Value: &Uint128{Hi: 11, Lo: 12}},
		To:     &AccountName{Value: &Uint128{Hi: 11, Lo: 12}},
		Amount: &Coin{Amount: &Safe64{Value: 100}},
		Memo:   "this is transfer",
	}

	return top
}

func TestVerifySig(t *testing.T) {

	sigKey, err := crypto.GenerateKey()

	if err != nil{
		fmt.Println("GenerateKey error")
		return
	}

	pubKey := crypto.FromECDSAPub( &sigKey.PublicKey )

	fmt.Println("ecc gen priv Key: ", len(crypto.FromECDSA(sigKey)), crypto.FromECDSA(sigKey) )
	fmt.Println("ecc gen pub Key: ", len(pubKey), pubKey )


	cid := ChainId{ Value:0 }
	strx := new(SignedTransaction)

	strx.Trx = new(Transaction)
	strx.Trx.RefBlockNum 		= 1
	strx.Trx.RefBlockPrefix		= 1
	strx.Trx.Expiration			= &TimePointSec{UtcSeconds:10}

	top := &Operation_Top{ Top: makeOp() }
	op1 := &Operation{ Op : top }
	strx.Trx.Operations = append( strx.Trx.Operations, op1 )

	res := strx.Sign( crypto.FromECDSA(sigKey) , cid)

	strx.Signatures = append(strx.Signatures, &SignatureType{ Sig:res } )

	fmt.Println("sign result: ", res, ": len: ", len(res) )

	fmt.Println( "VerifySig result: ", strx.VerifySig( pubKey , cid ) )


	expPubKey , _ := strx.ExportPubKey( cid)
	fmt.Println( "Export PubKey: ", expPubKey )
}