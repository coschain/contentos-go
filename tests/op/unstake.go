package op

import (
	"github.com/coschain/contentos-go/common/constants"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

type UnStakeTester struct {
	acc0, acc1, acc2 *DandelionAccount
}

var stakeAmount uint64 = 100

func (tester *UnStakeTester) Test(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	t.Run("normal", d.Test(tester.normal))
	t.Run("wrong creditor", d.Test(tester.wrongCreditor))
	t.Run("wrong debtor", d.Test(tester.wrongDebtor))
	t.Run("wrong creditor and debtor", d.Test(tester.wrongCreditorAndDebtor))
	t.Run("unStake when frozen", d.Test(tester.unStakeWhenFreeze))
	t.Run("unStake amount zero", d.Test(tester.unStakeAmountZero))
	t.Run("insufficient vest", d.Test(tester.insufficientVest))
	t.Run("no stake record", d.Test(tester.noStakeRecord))

}

func (tester *UnStakeTester) notStaked(t *testing.T, d *Dandelion) {

}


func (tester *UnStakeTester) normal(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	name0 := tester.acc0.Name
	name1 := tester.acc1.Name
	//Firstly stake
	balance0 := tester.acc0.GetBalance().Value
	stakeVest1 := tester.acc1.GetStakeVest().Value
	a.NoError(tester.acc0.SendTrx(Stake(name0, name1, stakeAmount)))
	a.NoError(d.ProduceBlocks(1))
	a.Equal(balance0-stakeAmount, tester.acc0.GetBalance().Value)
	a.Equal(stakeVest1+stakeAmount, tester.acc1.GetStakeVest().Value)
	//unStake
	//can only unStake after the stakeFreezeTime time
	a.NoError(d.ProduceBlocks(constants.StakeFreezeTime + 5))
	curBalance0 := tester.acc0.GetBalance().Value
	curStakeVest1 := tester.acc1.GetStakeVest().Value
	unStakeAmount := stakeAmount/2
	a.NoError(tester.acc0.SendTrx(UnStake(name0, name1, unStakeAmount)))
	a.NoError(d.ProduceBlocks(1))
	a.Equal(curBalance0+unStakeAmount, tester.acc0.GetBalance().Value)
	a.Equal(curStakeVest1-unStakeAmount, tester.acc1.GetStakeVest().Value)
}


func (tester *UnStakeTester) wrongCreditor(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	//account1 not exist
	creditor := d.Account("account1")
	a.Empty(creditor.CheckExist())

	name0 := tester.acc0.Name
	name1 := tester.acc1.Name
	balance0 := tester.acc0.GetBalance().Value
	stakeVest1 := tester.acc1.GetStakeVest().Value
	a.NoError(tester.acc0.SendTrx(Stake(name0, name1, stakeAmount)))
	a.NoError(d.ProduceBlocks(1))
	a.Equal(balance0-stakeAmount, tester.acc0.GetBalance().Value)
	a.Equal(stakeVest1+stakeAmount, tester.acc1.GetStakeVest().Value)
	a.NoError(d.ProduceBlocks(constants.StakeFreezeTime + 5))

	curStakeVest1 := tester.acc1.GetStakeVest().Value
	a.NotEmpty(curStakeVest1)
	amount := curStakeVest1/2
	if amount == 0 {
		amount = stakeVest1
	}
	a.Error(tester.acc2.SendTrx(UnStake(creditor.Name, name1, amount)))
	a.NoError(d.ProduceBlocks(1))
	a.Equal(curStakeVest1, tester.acc1.GetStakeVest().Value)
}

func (tester *UnStakeTester) wrongDebtor(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	//account2 not exist
	debtor := d.Account("account2")
	a.Empty(debtor.CheckExist())

	name0 := tester.acc0.Name
	name2 := tester.acc2.Name
	balance0 := tester.acc0.GetBalance().Value
	stakeVest2 := tester.acc2.GetStakeVest().Value
	a.NoError(tester.acc0.SendTrx(Stake(name0, name2, stakeAmount)))
	a.NoError(d.ProduceBlocks(1))
	a.Equal(balance0-stakeAmount, tester.acc0.GetBalance().Value)
	a.Equal(stakeVest2+stakeAmount, tester.acc2.GetStakeVest().Value)
	a.NoError(d.ProduceBlocks(constants.StakeFreezeTime + 5))

	curStakeVest2 := tester.acc2.GetStakeVest().Value
	curBalance0 := tester.acc0.GetBalance().Value
	amount := curStakeVest2/2
	if amount == 0 {
		amount = curStakeVest2
	}
	a.NoError(tester.acc0.SendTrx(UnStake(name0, debtor.Name, amount)))
	a.NoError(d.ProduceBlocks(1))
	a.Equal(curBalance0, tester.acc0.GetBalance().Value)
}

func (tester *UnStakeTester) wrongCreditorAndDebtor(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	//account3 not exist
	creditor := d.Account("account3")
	a.Empty(creditor.CheckExist())

	//account4 not exist
	debtor := d.Account("account4")
	a.Empty(debtor.CheckExist())
	a.NoError(d.ProduceBlocks(constants.StakeFreezeTime + 5))
	a.Error(tester.acc0.SendTrx(UnStake(creditor.Name, debtor.Name, stakeAmount)))
	a.NoError(d.ProduceBlocks(1))
	a.Empty(creditor.GetBalance())
	a.Empty(debtor.GetStakeVest())

}

func (tester *UnStakeTester) unStakeWhenFreeze(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	name0 := tester.acc0.Name
	name2 := tester.acc2.Name
	balance0 := tester.acc0.GetBalance().Value
	stakeVest2 := tester.acc2.GetStakeVest().Value
	a.NoError(tester.acc0.SendTrx(Stake(name0, name2, stakeAmount)))
	a.NoError(d.ProduceBlocks(1))
	a.Equal(balance0-stakeAmount, tester.acc0.GetBalance().Value)
	a.Equal(stakeVest2+stakeAmount, tester.acc2.GetStakeVest().Value)

	//unStake when frozen
	curStakeVest2 := tester.acc2.GetStakeVest().Value
	curBalance0 := tester.acc0.GetBalance().Value
	amount := curStakeVest2/2
	if amount == 0 {
		amount = curStakeVest2
	}
	a.NoError(tester.acc0.SendTrx(UnStake(name0, name2, amount)))
	a.NoError(d.ProduceBlocks(1))
	a.Equal(curBalance0, tester.acc0.GetBalance().Value)
	a.Equal(curStakeVest2, tester.acc2.GetStakeVest().Value)
}


func (tester *UnStakeTester) unStakeAmountZero(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	name0 := tester.acc0.Name
	name1 := tester.acc1.Name
	balance1 := tester.acc1.GetBalance().Value
	stakeVest0 := tester.acc0.GetStakeVest().Value
	a.NoError(tester.acc1.SendTrx(Stake(name1, name0, stakeAmount)))
	a.NoError(d.ProduceBlocks(1))
	a.Equal(balance1-stakeAmount, tester.acc1.GetBalance().Value)
	a.Equal(stakeVest0+stakeAmount, tester.acc0.GetStakeVest().Value)
	a.NoError(d.ProduceBlocks(constants.StakeFreezeTime + 5))

	curStakeVest0 := tester.acc0.GetStakeVest().Value
	curBalance1 := tester.acc1.GetBalance().Value
	a.Error(tester.acc1.SendTrx(UnStake(name1, name0, 0)))
	a.NoError(d.ProduceBlocks(1))
	a.Equal(curStakeVest0, tester.acc0.GetStakeVest().Value)
	a.Equal(curBalance1, tester.acc1.GetBalance().Value)

}


func (tester *UnStakeTester) insufficientVest(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	name0 := tester.acc0.Name
	name2 := tester.acc2.Name
	balance2 := tester.acc2.GetBalance().Value
	stakeVest0 := tester.acc0.GetStakeVest().Value
	a.NoError(tester.acc2.SendTrx(Stake(name2, name0, stakeAmount)))
	a.NoError(d.ProduceBlocks(1))
	a.Equal(balance2-stakeAmount, tester.acc2.GetBalance().Value)
	a.Equal(stakeVest0+stakeAmount, tester.acc0.GetStakeVest().Value)
	a.NoError(d.ProduceBlocks(constants.StakeFreezeTime + 5))

	curStakeVest0 := tester.acc0.GetStakeVest().Value
	curBalance2 := tester.acc2.GetBalance().Value
	//unStake amount greater than stake vest
	a.NoError(tester.acc2.SendTrx(UnStake(name2, name0, math.MaxUint64)))
	a.NoError(d.ProduceBlocks(1))
	a.Equal(curStakeVest0, tester.acc0.GetStakeVest().Value)
	a.Equal(curBalance2, tester.acc2.GetBalance().Value)

}

func (tester *UnStakeTester) noStakeRecord(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	name0 := tester.acc0.Name
	name3 := "actor3"
	name4 := "actor4"
    a.Empty(d.Account(name3).CheckExist())
	a.Empty(d.Account(name4).CheckExist())
	acct3 := d.Account(name3)
	priv3, _ := prototype.GenerateNewKey()
	pub3, _ := priv3.PubKey()
	a.NoError(tester.acc0.SendTrx(AccountCreate(name0, acct3.Name, pub3,10, "")))
	a.NoError(d.ProduceBlocks(1))
	acct4 := d.Account(name4)
	priv4, _ := prototype.GenerateNewKey()
	pub4, _ := priv4.PubKey()
	a.NoError(tester.acc0.SendTrx(AccountCreate(name0, acct4.Name, pub4, 10, "")))
	a.NoError(d.ProduceBlocks(1))
	a.NotEmpty(acct3.CheckExist())
	a.NotEmpty(acct4.CheckExist())
	d.PutAccount(acct3.Name, priv3)
	d.PutAccount(acct4.Name, priv4)
	//no stake recode from acc3 to acc4
	balance3 := acct3.GetBalance().Value
	stakeVest4 := acct4.GetStakeVest().Value
	a.Empty(d.StakeRecord(name3, name4).CheckExist())
	a.NoError(acct3.SendTrx(UnStake(name3, name4, stakeAmount)))
	a.NoError(d.ProduceBlocks(1))
	a.Equal(balance3, acct3.GetBalance().Value)
	a.Equal(stakeVest4, acct4.GetStakeVest().Value)
}