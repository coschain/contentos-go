package op

import (
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/stretchr/testify/assert"
	"math"
	"strings"
	"testing"
)

type TransferToVestTester struct {
	acc0, acc1, acc2 *DandelionAccount
}

func (tester *TransferToVestTester) Test(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	t.Run("normal to self", d.Test(tester.normalToSelf))
	t.Run("normal to other", d.Test(tester.normalToOther))
	t.Run("too mach", d.Test(tester.tooMuch))
	t.Run("to unknown", d.Test(tester.toUnknown))
	t.Run("wrong sender", d.Test(tester.wrongSender))
	t.Run("big memo", d.Test(tester.bigMemo))
}

func (tester *TransferToVestTester) normalToSelf(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	vest0 := tester.acc0.GetVest().Value
	vest1 := tester.acc1.GetVest().Value
	vest2 := tester.acc2.GetVest().Value
	balance0 := tester.acc0.GetBalance().Value
	balance1 := tester.acc1.GetBalance().Value
	balance2 := tester.acc2.GetBalance().Value


	a.NoError(tester.acc0.SendTrx(TransferToVest(tester.acc0.Name, tester.acc0.Name, 100000, "")))
	a.NoError(tester.acc1.SendTrx(TransferToVest(tester.acc1.Name, tester.acc1.Name, 100001, "")))
	a.NoError(tester.acc2.SendTrx(TransferToVest(tester.acc2.Name, tester.acc2.Name, 100002, "")))
	a.NoError(d.ProduceBlocks(1))

	a.Equal(balance0 - 100000, tester.acc0.GetBalance().Value)
	a.Equal(balance1 - 100001, tester.acc1.GetBalance().Value)
	a.Equal(balance2 - 100002, tester.acc2.GetBalance().Value)
	a.Equal(vest0+ 100000, tester.acc0.GetVest().Value)
	a.Equal(vest1+ 100001, tester.acc1.GetVest().Value)
	a.Equal(vest2+ 100002, tester.acc2.GetVest().Value)
}

func (tester *TransferToVestTester) normalToOther(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	vest0 := tester.acc0.GetVest().Value
	vest1 := tester.acc1.GetVest().Value
	vest2 := tester.acc2.GetVest().Value
	balance0 := tester.acc0.GetBalance().Value
	balance1 := tester.acc1.GetBalance().Value
	balance2 := tester.acc2.GetBalance().Value


	a.NoError(tester.acc0.SendTrx(TransferToVest(tester.acc0.Name, tester.acc1.Name, 100000, "")))
	a.NoError(tester.acc0.SendTrx(TransferToVest(tester.acc0.Name, tester.acc2.Name, 100001, "")))
	a.NoError(tester.acc2.SendTrx(TransferToVest(tester.acc2.Name, tester.acc1.Name, 100002, "")))
	a.NoError(d.ProduceBlocks(1))

	a.Equal(balance0 - 200001, tester.acc0.GetBalance().Value)
	a.Equal(balance1, tester.acc1.GetBalance().Value)
	a.Equal(balance2 - 100002, tester.acc2.GetBalance().Value)
	a.Equal(vest0, tester.acc0.GetVest().Value)
	a.Equal(vest1 + 200002, tester.acc1.GetVest().Value)
	a.Equal(vest2 + 100001, tester.acc2.GetVest().Value)
}

func (tester *TransferToVestTester) tooMuch(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	balance0 := tester.acc0.GetBalance().Value
	balance1 := tester.acc1.GetBalance().Value

	a.NoError(tester.acc0.SendTrx(TransferToVest(tester.acc0.Name, tester.acc1.Name, balance0 + 1, "")))
	a.NoError(tester.acc0.SendTrx(TransferToVest(tester.acc0.Name, tester.acc1.Name, math.MaxUint64, "")))
	a.NoError(d.ProduceBlocks(1))

	a.Equal(balance0, tester.acc0.GetBalance().Value)
	a.Equal(balance1, tester.acc1.GetBalance().Value)
}

func (tester *TransferToVestTester) toUnknown(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	balance0 := tester.acc0.GetBalance().Value
	a.NoError(tester.acc0.SendTrx(TransferToVest(tester.acc0.Name, "noexist1", 10, "")))
	a.NoError(tester.acc0.SendTrx(TransferToVest(tester.acc0.Name, "noexist2", 20, "")))
	a.NoError(tester.acc0.SendTrx(TransferToVest(tester.acc0.Name, "noexist3", 30, "")))
	a.NoError(d.ProduceBlocks(1))

	a.Equal(balance0, tester.acc0.GetBalance().Value)
}

func (tester *TransferToVestTester) wrongSender(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	balance0 := tester.acc0.GetBalance().Value
	balance1 := tester.acc1.GetBalance().Value
	balance2 := tester.acc2.GetBalance().Value

	a.Error(tester.acc0.SendTrx(TransferToVest(tester.acc1.Name, tester.acc2.Name, 10, "")))
	a.Error(tester.acc1.SendTrx(TransferToVest(tester.acc0.Name, tester.acc1.Name, 10, "")))
	a.Error(tester.acc2.SendTrx(TransferToVest(tester.acc1.Name, tester.acc2.Name, 10, "")))
	a.NoError(d.ProduceBlocks(1))

	a.Equal(balance0, tester.acc0.GetBalance().Value)
	a.Equal(balance1, tester.acc1.GetBalance().Value)
	a.Equal(balance2, tester.acc2.GetBalance().Value)
}

func (tester *TransferToVestTester) bigMemo(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.Error(tester.acc0.SendTrx(TransferToVest(tester.acc0.Name, tester.acc1.Name, 1, strings.Repeat("A", 4500))))
	a.NoError(tester.acc0.SendTrx(TransferToVest(tester.acc0.Name, tester.acc1.Name, 1, strings.Repeat("A", 4000))))
}
