package op

import (
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
	"github.com/stretchr/testify/assert"
	"testing"
)

type AccountUpdateTester struct {
	acc0, acc1, acc2 *DandelionAccount
}

func (tester *AccountUpdateTester) Test(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")


	t.Run("normal", d.Test(tester.normal))
	t.Run("update not exist account", d.Test(tester.wrongAccount))
	t.Run("public key too short", d.Test(tester.pubKeyTooShort))
	t.Run("public key too long", d.Test(tester.pubKeyTooLong))
	t.Run("update duplicate public Key", d.Test(tester.duplicatePublicKey))

}

func (tester *AccountUpdateTester) normal(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	priv, _ := prototype.GenerateNewKey()
	pub, _ := priv.PubKey()
	oldPub1 := tester.acc1.GetPubKey()
	a.NoError(tester.acc1.SendTrx(AccountUpdate(tester.acc1.Name, pub)))
	a.NoError(d.ProduceBlocks(1))
	newPub := tester.acc1.GetPubKey()
	a.Equal(newPub.Data, pub.Data)
	a.NotEqual(oldPub1.Data, newPub.Data)

}


func (tester *AccountUpdateTester) wrongAccount(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	priv, _ := prototype.GenerateNewKey()
	pub, _ := priv.PubKey()


	acctName := "account1"
	acct := d.Account(acctName)
	a.Empty(acct.CheckExist())
	a.Error(tester.acc1.SendTrx(AccountUpdate(acctName, pub)))
	a.NoError(d.ProduceBlocks(1))
	a.Empty(acct.GetPubKey())
}


func (tester *AccountUpdateTester) pubKeyTooShort(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	priv, _ := prototype.GenerateNewKey()
	pub, _ := priv.PubKey()
	oldPub0 := tester.acc0.GetPubKey()
	data := pub.Data[:10]
	newPub := &prototype.PublicKeyType{Data: data}
	a.Error(tester.acc0.SendTrx(AccountUpdate(tester.acc0.Name, newPub)))
	a.NoError(d.ProduceBlocks(1))
	a.Equal(tester.acc0.GetPubKey().Data, oldPub0.Data)
}

func (tester *AccountUpdateTester) pubKeyTooLong(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	priv, _ := prototype.GenerateNewKey()
	pub, _ := priv.PubKey()
	oldPub1 := tester.acc1.GetPubKey()
	data := pub.Data[:]
	data = append(data, []byte("public key is too long")...)
	newPub := &prototype.PublicKeyType{Data: data}
	a.Error(tester.acc1.SendTrx(AccountUpdate(tester.acc1.Name, newPub)))
	a.NoError(d.ProduceBlocks(1))
	a.Equal(oldPub1.Data, tester.acc1.GetPubKey().Data)

}

func (tester *AccountUpdateTester) duplicatePublicKey(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	pub2 := tester.acc2.GetPubKey()
	oldPub0 := tester.acc0.GetPubKey()

	a.NoError(tester.acc0.SendTrx(AccountUpdate(tester.acc0.Name, pub2)))
	a.NoError(d.ProduceBlocks(1))
	a.Equal(oldPub0.Data, tester.acc0.GetPubKey().Data)
}
