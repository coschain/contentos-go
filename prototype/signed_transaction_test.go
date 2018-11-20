package prototype

import "testing"
import (
	"fmt"
)

func makeOp() *TransferOperation {
	top := &TransferOperation{
		From:   &AccountName{Value: "alice"},
		To:     &AccountName{Value: "bob"},
		Amount: NewCoin(100),
		Memo:   "this is transfer",
	}

	return top
}

func TestVerifySig(t *testing.T) {

	sigKey, err := GenerateNewKey()

	if err != nil {
		fmt.Println("GenerateKey error")
		t.FailNow()
		return
	}

	pubKey, err := sigKey.PubKey()

	if err != nil {
		t.FailNow()
		return
	}

	fmt.Println("ecc gen priv Key: ", len(sigKey.Data), sigKey.Data)
	fmt.Println("ecc gen pub Key: ", len(pubKey.Data), pubKey.Data)

	strPrivWIF := sigKey.ToWIF()
	fmt.Println("PrivateKeyWIF:", strPrivWIF)
	sigKey2, err := PrivateKeyFromWIF(strPrivWIF)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
		return
	}
	if !sigKey.Equal(sigKey2) {
		fmt.Println("error wif convert")
		t.FailNow()
		return
	}

	strPubWIF := pubKey.ToWIF()
	fmt.Println("PublicKeyWIF:", strPubWIF)
	pubKey2, err := PublicKeyFromWIF(strPubWIF)

	if err != nil {
		fmt.Println(err)
		t.FailNow()
		return
	}
	if !pubKey.Equal(pubKey2) {
		fmt.Println("error wif convert")
		t.FailNow()
		return
	}

	cid := ChainId{Value: 0}

	strx := new(SignedTransaction)

	strx.Trx = new(Transaction)
	strx.Trx.RefBlockNum = 1
	strx.Trx.RefBlockPrefix = 1
	strx.Trx.Expiration = &TimePointSec{UtcSeconds: 10}

	strx.Trx.AddOperation(makeOp())

	res := strx.Sign(sigKey, cid)

	strx.Signatures = append(strx.Signatures, &SignatureType{Sig: res})

	fmt.Println("sign result: ", res, ": len: ", len(res))

	if !strx.VerifySig(pubKey, cid) {
		t.FailNow()
		return
	}
	fmt.Println("VerifySig result success")

	expPubKeys, err := strx.ExportPubKeys(cid)

	if err != nil {
		fmt.Println("ExportPubKeys failed")
		t.FailNow()
		return
	}

	for _, expPubKey := range expPubKeys {
		fmt.Println("Export PubKeys: ", expPubKey.ToWIF())
	}
}
