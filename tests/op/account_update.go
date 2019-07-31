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
	t.Run("new private key work", d.Test(tester.verifyNewPriKeyWork))
	t.Run("old private key not work", d.Test(tester.verifyOldPriKeyNotWork))

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
	//save new private key
	d.PutAccount(tester.acc1.Name, priv)

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

func (tester *AccountUpdateTester) verifyNewPriKeyWork(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	name0 := tester.acc0.Name
	name2 := tester.acc2.Name
	priv, _ := prototype.GenerateNewKey()
	pub, _ := priv.PubKey()
	oldPub0 := tester.acc0.GetPubKey()
	a.NoError(tester.acc0.SendTrx(AccountUpdate(name0, pub)))
	a.NoError(d.ProduceBlocks(1))
	newPub := tester.acc0.GetPubKey()
	a.Equal(newPub.Data, pub.Data)
	a.NotEqual(oldPub0.Data, newPub.Data)
	//save new private key
	d.PutAccount(name0, priv)
	//transfer and use actor0's new new private key to sign to verify actor0's
	// new private key work after updating public key
	balance0 := tester.acc0.GetBalance().Value
	balance2 := tester.acc2.GetBalance().Value
	var amount uint64 = 10
	a.NoError(tester.acc0.SendTrx(Transfer(name0, name2, amount, "")))
	a.NoError(d.ProduceBlocks(1))
    a.Equal(balance0-amount, tester.acc0.GetBalance().Value)
	a.Equal(balance2+amount, tester.acc2.GetBalance().Value)
}

func (tester *AccountUpdateTester) verifyOldPriKeyNotWork(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	name0 := tester.acc0.Name
	name2 := tester.acc2.Name
	priv, _ := prototype.GenerateNewKey()
	pub, _ := priv.PubKey()
	oldPub2 := tester.acc2.GetPubKey()
	a.NoError(tester.acc2.SendTrx(AccountUpdate(name2, pub)))
	a.NoError(d.ProduceBlocks(1))
	newPub := tester.acc2.GetPubKey()
	a.Equal(newPub.Data, pub.Data)
	a.NotEqual(oldPub2.Data, newPub.Data)
	//transfer and use actor2 old private key to sign to verify actor2's old private key
	// not work after updating public key
	balance0 := tester.acc0.GetBalance().Value
	balance2 := tester.acc2.GetBalance().Value
	var amount uint64 = 10
	//old private key signature verification failed if use new public key, so transfer will fail
	a.Error(tester.acc2.SendTrx(Transfer(name2, name0, amount, "")))
	a.NoError(d.ProduceBlocks(1))
	a.Equal(balance0, tester.acc0.GetBalance().Value)
	a.Equal(balance2, tester.acc2.GetBalance().Value)
}