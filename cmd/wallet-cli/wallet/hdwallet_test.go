package wallet

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

type BIP39 struct {
	Mnemonic string `json:"Mnemonic"`
	PublicKey string `json:"PublicKey"`
	PrivateKey string `json:"PrivateKey"`
}

func TestBaseHDWallet_GenerateFromMnemonic(t *testing.T) {
	data, _ := ioutil.ReadFile("testdata/bip39.json")
	a := assert.New(t)
	wallet := NewBaseHDWallet("1", "")
	var bips []BIP39
	_ = json.Unmarshal(data, &bips)
	for _, bip := range bips {
		public, private, err := wallet.GenerateFromMnemonic(bip.Mnemonic)
		a.NoError(err, "generate from mnemonic error!", bip.Mnemonic, err)
		a.Equal(public, bip.PublicKey)
		a.Equal(private, bip.PrivateKey)
	}
}
