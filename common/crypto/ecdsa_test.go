package crypto

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
)

func TestECDSA(t *testing.T) {
	privKey, err := GenerateKey()
	if err != nil {
		t.Errorf("failed to generate key: %s\n", err.Error())
	}
	privKeyStr, _ := privKey.ToString()
	fmt.Println("generate private key:\n", privKeyStr)

	pubKey := privKey.Public()
	pubKeyStr, _ := pubKey.ToString()
	fmt.Println("corresponding public key:\n", pubKeyStr)

	// reconstruct the key pair using privKeyStr and pubKeyStr
	privKey1, err := ConstructKeyFromString(privKeyStr)
	if err != nil {
		t.Errorf("failed to reconstruct private key: %s\n", err.Error())
	}
	privKey1Str, _ := privKey1.ToString()
	if strings.Compare(privKeyStr, privKey1Str) != 0 {
		t.Errorf("mismatched private key string")
	}
	pubKey1, err := ConstructPubKeyFromString(pubKeyStr)
	if err != nil {
		t.Errorf("failed to reconstruct public key: %s\n", err.Error())
	}
	pubKey1Str, _ := pubKey1.ToString()
	if strings.Compare(pubKeyStr, pubKey1Str) != 0 {
		t.Errorf("mismatched private key string")
	}

	// sign
	message := "hello world!"
	digest := sha256.Sum256([]byte(message))
	sig, err := privKey.Sign(digest[:])
	if err != nil {
		t.Error("failed to sign digest: ", err.Error())
	}
	if pubKey.Verify(digest[:], sig) != true {
		t.Error("failed to verify signature")
	}
}
