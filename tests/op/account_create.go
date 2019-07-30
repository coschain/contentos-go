package op

import (
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

type AccountCreateTester struct {
	acc0, acc1, acc2 *DandelionAccount
}

func (tester *AccountCreateTester) Test(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	t.Run("normal", d.Test(tester.normal))
	t.Run("created account name too long", d.Test(tester.acctNameTooLong))
	t.Run("created account name too short", d.Test(tester.acctNameTooShort))
	t.Run("creator insufficient Balance", d.Test(tester.insufficientBalance))
	t.Run("creator not exist", d.Test(tester.wrongCreator))
	t.Run("duplicate public key", d.Test(tester.duplicatePublicKey))
	t.Run("illegal character format", d.Test(tester.illegalCharacterFormat))
	t.Run("create fee too low", d.Test(tester.feeTooLow))


}

func (tester *AccountCreateTester) normal(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	priv, _ := prototype.GenerateNewKey()
	pub, _ := priv.PubKey()

	acctName := "actor3"
	balance0 := tester.acc0.GetBalance().Value
	a.NoError(tester.acc0.SendTrx(AccountCreate(tester.acc0.Name, acctName, pub, 10,  "")))
	a.NoError(d.ProduceBlocks(1))
	a.Equal(balance0-10, tester.acc0.GetBalance().Value)
	newAcct := d.Account(acctName)
	a.NotEmpty(newAcct.CheckExist())

}

func (tester *AccountCreateTester) acctNameTooLong(t *testing.T, d *Dandelion)  {
	a := assert.New(t)
	priv, _ := prototype.GenerateNewKey()
	pub, _ := priv.PubKey()

	//the length of new account name can't exceed 16
	lName := "testTooLongAccount"
	balance0 := tester.acc0.GetBalance().Value
	a.Error(tester.acc0.SendTrx(AccountCreate(tester.acc0.Name, lName ,pub, 10, "")))
	a.NoError(d.ProduceBlocks(1))
	a.Equal(balance0, tester.acc0.GetBalance().Value)
	newAcct := d.Account(lName)
	a.Empty(newAcct.CheckExist())

}

func (tester *AccountCreateTester) acctNameTooShort(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	priv, _ := prototype.GenerateNewKey()
	pub, _ := priv.PubKey()

	//the length of new account name can't less than 6
	sName := "acct"
	balance0 := tester.acc0.GetBalance().Value
	a.Error(tester.acc0.SendTrx(AccountCreate(tester.acc0.Name, sName ,pub, 10, "")))
	a.NoError(d.ProduceBlocks(1))
	a.Equal(balance0, tester.acc0.GetBalance().Value)
	newAcct := d.Account(sName)
	a.Empty(newAcct.CheckExist())
}

func (tester *AccountCreateTester) insufficientBalance(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	priv, _ := prototype.GenerateNewKey()
	pub, _ := priv.PubKey()

	acctName := "account1"
	balance1 := tester.acc1.GetBalance().Value
	a.NoError(tester.acc1.SendTrx(AccountCreate(tester.acc1.Name, acctName ,pub, math.MaxUint64, "")))
	a.NoError(d.ProduceBlocks(1))
    a.Equal(balance1, tester.acc1.GetBalance().Value)
	newAcct := d.Account(acctName)
	a.Empty(newAcct.CheckExist())
}

func (tester *AccountCreateTester) wrongCreator(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	priv, _ := prototype.GenerateNewKey()
	pub, _ := priv.PubKey()

	//the creator must already exist
	acctName := "account2"
	creator := "testAccount"
	a.Error(tester.acc2.SendTrx(AccountCreate(creator, acctName, pub, 10, "")))
	a.NoError(d.ProduceBlocks(1))
	newAcct := d.Account(acctName)
	a.Empty(newAcct.CheckExist())
}

func (tester *AccountCreateTester) duplicatePublicKey(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	pub0 := tester.acc0.GetPubKey()

	acctName := "account3"
	balance0 := tester.acc0.GetBalance().Value
	a.NoError(tester.acc0.SendTrx(AccountCreate(tester.acc0.Name, acctName, pub0, 10, "")))
	a.NoError(d.ProduceBlocks(1))
	acct := d.Account(acctName)
	a.Equal(balance0, tester.acc0.GetBalance().Value)
	a.Empty(acct.CheckExist())

}


func (tester *AccountCreateTester) illegalCharacterFormat(t *testing.T, d *Dandelion){
	a := assert.New(t)
	priv, _ := prototype.GenerateNewKey()
	pub, _ := priv.PubKey()

	//Account name consists only of numbers and uppercase and lowercase letters
	acctName := "account_4"
	balance0 := tester.acc0.GetBalance().Value
	a.Error(tester.acc0.SendTrx(AccountCreate(tester.acc0.Name, acctName, pub, 10, "")))
	a.NoError(d.ProduceBlocks(1))
	a.Equal(balance0, tester.acc0.GetBalance().Value)
	newAcct := d.Account(acctName)
	a.Empty(newAcct.CheckExist())
}


func (tester *AccountCreateTester) feeTooLow(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	priv, _ := prototype.GenerateNewKey()
	pub, _ := priv.PubKey()

	d.TrxPool().ModifyProps(func(oldProps *prototype.DynamicProperties) {
		oldProps.AccountCreateFee = &prototype.Coin{Value:10}
	})
	acctName := "account5"
	balance2 := tester.acc2.GetBalance().Value
	a.NoError(tester.acc2.SendTrx(AccountCreate(tester.acc2.Name, acctName, pub, 1, "")))
	a.NoError(d.ProduceBlocks(1))
	a.Equal(balance2, tester.acc2.GetBalance().Value)
	newAcct := d.Account(acctName)
	a.Empty(newAcct.CheckExist())
}
