package dandelion

import (
	"github.com/coschain/contentos-go/prototype"
	"testing"
)

func TestRedDandelion_CreateAccount(t *testing.T) {
	dandelion, err := NewRedDandelion()
	if err != nil {
		t.Error(err)
	}
	err = dandelion.OpenDatabase()
	if err != nil {
		t.Error(err)
	}
	defaultPrivKey, err := prototype.GenerateNewKeyFromBytes([]byte(initPrivKey))
	if err != nil {
		t.Error(err)
	}
	defaultPubKey, err := defaultPrivKey.PubKey()
	if err != nil {
		t.Error(err)
	}

	keys := prototype.NewAuthorityFromPubKey(defaultPubKey)

	// create account with default pub key
	acop := &prototype.AccountCreateOperation{
		Fee:            prototype.NewCoin(1),
		Creator:        &prototype.AccountName{Value: "initminer"},
		NewAccountName: &prototype.AccountName{Value: "kochiya"},
		Owner:          keys,
		Posting:        keys,
		Active:         keys,
	}
	// use initminer's priv key sign
	signTx, err := dandelion.Sign(defaultPrivKey.ToWIF(), acop)
	if err != nil {
		t.Error(err)
	}
	dandelion.PushTrx(signTx)
	dandelion.GenerateBlock()
	if err != nil {
		t.Error(err)
	}
	defer func() {
		err := dandelion.Clean()
		if err != nil {
			t.Error(err)
		}
	}()
}
