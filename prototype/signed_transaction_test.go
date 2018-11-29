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

func makeFixKeyMap() map[string]string  {
	fixKeys := make(map[string]string)

	fixKeys["123456"] = "2AvYqihDZjq7pFeZNuBYjBW1hQyPUw36xZB25g8UYfRLKwh7k9"
	fixKeys["initminer"] = "28PFDCwkDWNFYSeFEyN5mct1J75v5ZxwpVtAb3mb3XySJBrGSj"
	fixKeys["initminer1"] = "2su2nYzmkfT7p1JbiStegUN3Prrkr36p6CPQSvGG3TmRbEVEqy"
	fixKeys["initminer2"] = "bM8zkJXxvdfyKCweWZaT6vgEPCtWCEX3S4EspmiiSjwgRzgcF"
	fixKeys["1"] = "2CL5gdFyX4XF4sq6yoxPBpX92xHtnyz7K5JG9gGSKDzqmzgyzp"

	return fixKeys
}

func TestFixGenKey(t *testing.T) {
	fixKeys := makeFixKeyMap()
	for k,v:= range fixKeys {
		res1, err := GenerateNewKeyFromBytes( []byte(k) )

		if err != nil{
			t.Fatal(err)
		}

		if res1.ToWIF() != v{
			t.Fatal("key Error")
		}

		res2, err := GenerateNewKeyFromBytes( []byte(k) )

		if err != nil{
			t.Fatal(err)
		}

		if !res1.Equal( res2 ){
			t.Fatal("key Error")
		}
	}
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
