package op

import (
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

type StakeTester struct {
	acc0, acc1, acc2 *DandelionAccount
}

func (tester *StakeTester) Test(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	t.Run("normal", d.Test(tester.normal))
	t.Run("stake wrong sender", d.Test(tester.wrongSender))
	t.Run("stake wrong receiver", d.Test(tester.wrongReceiver))
	t.Run("stake wrong sender and receiver", d.Test(tester.wrongSenderAndReceiver))
	t.Run("stake zero", d.Test(tester.amountZero))
	t.Run("stake insufficient Balance", d.Test(tester.insufficientBalance))
	t.Run("multiple Stake", d.Test(tester.multipleStake))
}


func (tester *StakeTester) normal(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	balance0 := tester.acc0.GetBalance().Value
	stakeVest1 := tester.acc1.GetStakeVesting().Value
	a.NoError(tester.acc0.SendTrx(Stake(tester.acc0.Name, tester.acc1.Name, 100)))
	a.NoError(d.ProduceBlocks(1))
	a.Equal(balance0-100, tester.acc0.GetBalance().Value)
	a.Equal(stakeVest1+100, tester.acc1.GetStakeVesting().Value)
}

func (tester *StakeTester) wrongSender(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	sender := d.Account("account1")
	a.Empty(sender.CheckExist())
	stakeVest2 := tester.acc2.GetStakeVesting().Value
	a.Error(tester.acc0.SendTrx(Stake(sender.Name, tester.acc2.Name, 1)))
	a.NoError(d.ProduceBlocks(1))
	a.Equal(stakeVest2, tester.acc2.GetStakeVesting().Value)
}


func (tester *StakeTester) wrongReceiver(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	receiver := d.Account("account2")
	a.Empty(receiver.CheckExist())
	balance1 := tester.acc1.GetBalance().Value
	a.Error(tester.acc0.SendTrx(Stake(tester.acc1.Name, receiver.Name, 1)))
	a.NoError(d.ProduceBlocks(1))
	a.Equal(balance1, tester.acc1.GetBalance().Value)
}

func (tester *StakeTester) wrongSenderAndReceiver(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	sender := d.Account("account3")
	a.Empty(sender.CheckExist())
	receiver := d.Account("account4")
	a.Empty(receiver.CheckExist())
	a.Error(tester.acc2.SendTrx(Stake(sender.Name, receiver.Name, 10)))
	a.NoError(d.ProduceBlocks(1))
	a.Nil(sender.GetBalance())
	a.Nil(receiver.GetStakeVesting())

}

func (tester *StakeTester) amountZero(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	balance0 := tester.acc0.GetBalance().Value
	stakeVest2 := tester.acc2.GetStakeVesting().Value
	a.Error(tester.acc1.SendTrx(Stake(tester.acc0.Name, tester.acc2.Name, 0)))
	a.Equal(balance0, tester.acc0.GetBalance().Value)
	a.Equal(stakeVest2, tester.acc2.GetStakeVesting().Value)

}

func (tester *StakeTester) insufficientBalance(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	balance2 := tester.acc2.GetBalance().Value
	stakeVest0 := tester.acc0.GetStakeVesting().Value
	a.NoError(tester.acc2.SendTrx(Stake(tester.acc2.Name, tester.acc0.Name, math.MaxUint64)))
	a.NoError(d.ProduceBlocks(1))
	a.Equal(balance2, tester.acc2.GetBalance().Value)
	a.Equal(stakeVest0, tester.acc0.GetStakeVesting().Value)

}

func (tester *StakeTester) multipleStake(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	acctList := []*DandelionAccount{tester.acc0, tester.acc1, tester.acc2}
	listLen := len(acctList)

	for i := 0; i < 6; i++ {
		balance0 := tester.acc0.GetBalance().Value
		acct := acctList[i%listLen]
		stakeVest := acct.GetStakeVesting().Value
		amount := uint64(20*(i+1))
		a.NoError(tester.acc0.SendTrx(Stake(tester.acc0.Name, acct.Name, amount)))
		a.NoError(d.ProduceBlocks(1))
		a.Equal(balance0-amount, tester.acc0.GetBalance().Value)
		a.Equal(stakeVest+amount, acct.GetStakeVesting().Value)
	}

}


