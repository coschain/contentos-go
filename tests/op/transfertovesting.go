package op

import (
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

type TransferToVestingTester struct {
	acc0, acc1, acc2 *DandelionAccount
}

func (tester *TransferToVestingTester) Test(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	t.Run("normal to self", d.Test(tester.normalToSelf))
	t.Run("normal to other", d.Test(tester.normalToOther))
	t.Run("too mach", d.Test(tester.tooMuch))
	t.Run("to unknown", d.Test(tester.toUnknown))
	t.Run("wrong sender", d.Test(tester.wrongSender))
}

func (tester *TransferToVestingTester) normalToSelf(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	vestingShares0 := tester.acc0.GetVestingShares().Value
	vestingShares1 := tester.acc1.GetVestingShares().Value
	vestingShares2 := tester.acc2.GetVestingShares().Value
	balance0 := tester.acc0.GetBalance().Value
	balance1 := tester.acc1.GetBalance().Value
	balance2 := tester.acc2.GetBalance().Value


	a.NoError(tester.acc0.SendTrx(TransferToVesting(tester.acc0.Name, tester.acc0.Name, 100000)))
	a.NoError(tester.acc1.SendTrx(TransferToVesting(tester.acc1.Name, tester.acc1.Name, 100001)))
	a.NoError(tester.acc2.SendTrx(TransferToVesting(tester.acc2.Name, tester.acc2.Name, 100002)))
	a.NoError(d.ProduceBlocks(1))

	a.Equal(balance0 - 100000, tester.acc0.GetBalance().Value)
	a.Equal(balance1 - 100001, tester.acc1.GetBalance().Value)
	a.Equal(balance2 - 100002, tester.acc2.GetBalance().Value)
	a.Equal(vestingShares0 + 100000, tester.acc0.GetVestingShares().Value)
	a.Equal(vestingShares1 + 100001, tester.acc1.GetVestingShares().Value)
	a.Equal(vestingShares2 + 100002, tester.acc2.GetVestingShares().Value)
}

func (tester *TransferToVestingTester) normalToOther(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	vestingShares0 := tester.acc0.GetVestingShares().Value
	vestingShares1 := tester.acc1.GetVestingShares().Value
	vestingShares2 := tester.acc2.GetVestingShares().Value
	balance0 := tester.acc0.GetBalance().Value
	balance1 := tester.acc1.GetBalance().Value
	balance2 := tester.acc2.GetBalance().Value


	a.NoError(tester.acc0.SendTrx(TransferToVesting(tester.acc0.Name, tester.acc1.Name, 100000)))
	a.NoError(tester.acc0.SendTrx(TransferToVesting(tester.acc0.Name, tester.acc2.Name, 100001)))
	a.NoError(tester.acc2.SendTrx(TransferToVesting(tester.acc2.Name, tester.acc1.Name, 100002)))
	a.NoError(d.ProduceBlocks(1))

	a.Equal(balance0 - 200001, tester.acc0.GetBalance().Value)
	a.Equal(balance1, tester.acc1.GetBalance().Value)
	a.Equal(balance2 - 100002, tester.acc2.GetBalance().Value)
	a.Equal(vestingShares0, tester.acc0.GetVestingShares().Value)
	a.Equal(vestingShares1 + 200002, tester.acc1.GetVestingShares().Value)
	a.Equal(vestingShares2 + 100001, tester.acc2.GetVestingShares().Value)
}

func (tester *TransferToVestingTester) tooMuch(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	balance0 := tester.acc0.GetBalance().Value
	balance1 := tester.acc1.GetBalance().Value

	a.NoError(tester.acc0.SendTrx(TransferToVesting(tester.acc0.Name, tester.acc1.Name, balance0 + 1)))
	a.NoError(tester.acc0.SendTrx(TransferToVesting(tester.acc0.Name, tester.acc1.Name, math.MaxUint64)))
	a.NoError(d.ProduceBlocks(1))

	a.Equal(balance0, tester.acc0.GetBalance().Value)
	a.Equal(balance1, tester.acc1.GetBalance().Value)
}

func (tester *TransferToVestingTester) toUnknown(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	balance0 := tester.acc0.GetBalance().Value
	a.NoError(tester.acc0.SendTrx(TransferToVesting(tester.acc0.Name, "NONEXIST1", 10)))
	a.NoError(tester.acc0.SendTrx(TransferToVesting(tester.acc0.Name, "NONEXIST2", 20)))
	a.NoError(tester.acc0.SendTrx(TransferToVesting(tester.acc0.Name, "NONEXIST3", 30)))
	a.NoError(d.ProduceBlocks(1))

	a.Equal(balance0, tester.acc0.GetBalance().Value)
}

func (tester *TransferToVestingTester) wrongSender(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	balance0 := tester.acc0.GetBalance().Value
	balance1 := tester.acc1.GetBalance().Value
	balance2 := tester.acc2.GetBalance().Value

	a.Error(tester.acc0.SendTrx(TransferToVesting(tester.acc1.Name, tester.acc2.Name, 10)))
	a.Error(tester.acc1.SendTrx(TransferToVesting(tester.acc0.Name, tester.acc1.Name, 10)))
	a.Error(tester.acc2.SendTrx(TransferToVesting(tester.acc1.Name, tester.acc2.Name, 10)))
	a.NoError(d.ProduceBlocks(1))

	a.Equal(balance0, tester.acc0.GetBalance().Value)
	a.Equal(balance1, tester.acc1.GetBalance().Value)
	a.Equal(balance2, tester.acc2.GetBalance().Value)
}
