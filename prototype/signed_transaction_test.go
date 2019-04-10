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

func makeFixKeyMap() map[string]string {
	fixKeys := make(map[string]string)

	fixKeys["123456"] = "47fj5Aj4zR4FdqMTxXBW3nGvp2h3BmutbdHGEN2UCfopx1fZtZ"
	fixKeys["initminer"] = "458RSeybeBbPN2M9pbN36DevQioZnQqjTZzQjjfahYMvvFpib9"
	fixKeys["initminer1"] = "4peD212dBLgFdb1WJ4tc158xWUaQYsybjGVebcAFhU9vDJTgzP"
	fixKeys["initminer2"] = "3Y6KECLPPbronturE8ZXmhhbLzvYD37JgWAV2WfiNT8SJVy8kf"
	fixKeys["1"] = "495Fv5HpwjkNtTY1ZRxLWRJ49a1YVprtx9QWJNARyEPLQ4fS9E"

	return fixKeys
}

func TestFixGenKey(t *testing.T) {
	fixKeys := makeFixKeyMap()
	for k, v := range fixKeys {
		res1, err := GenerateNewKeyFromBytes([]byte(k))

		if err != nil {
			t.Fatal(err)
		}

		if res1.ToWIF() != v {
			t.Fatal("key Error")
		}

		res2, err := GenerateNewKeyFromBytes([]byte(k))

		if err != nil {
			t.Fatal(err)
		}

		if !res1.Equal(res2) {
			t.Fatal("key Error")
		}
	}
}

// test for private keys that start with 0x00's
func TestVerifySig_00(t *testing.T) {
	sigKey := &PrivateKeyType{
		Data: []byte{ 0, 194, 14, 189, 29, 16, 93, 75, 11, 144, 186, 152, 74, 222, 8, 40, 249, 115, 66, 160, 178, 41, 67, 235, 31, 9, 213, 64, 41, 148, 218, 181 },
	}
	sigKey2, err := PrivateKeyFromWIF(sigKey.ToWIF())
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	if !sigKey.Equal(sigKey2) {
		fmt.Println("error wif convert")
		t.FailNow()
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

	strx.Signature =  &SignatureType{Sig: res}

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

	//for _, expPubKey := range expPubKeys {
	//	fmt.Println("Export PubKeys: ", expPubKey.ToWIF())
	//}
	fmt.Println("Export PubKeys: ", expPubKeys.ToWIF())
}
