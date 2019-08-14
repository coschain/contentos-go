package op

import (
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

type TransferTester struct {
	acc0, acc1, acc2 *DandelionAccount
}

func (tester *TransferTester) Test(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	t.Run("normal", d.Test(tester.normal))
	t.Run("too-much", d.Test(tester.tooMuch))
	t.Run("to-unknown", d.Test(tester.toUnknown))
	t.Run("to-self", d.Test(tester.toSelf))
	t.Run("wrong-sender", d.Test(tester.wrongSender))
}

func (tester *TransferTester) normal(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	balance0 := tester.acc0.GetBalance().Value
	balance1 := tester.acc1.GetBalance().Value
	balance2 := tester.acc2.GetBalance().Value

	a.NoError(tester.acc0.SendTrx(Transfer(tester.acc0.Name, tester.acc1.Name, 10, "")))
	a.NoError(tester.acc0.SendTrx(Transfer(tester.acc0.Name, tester.acc2.Name, 30, "")))
	a.NoError(tester.acc2.SendTrx(Transfer(tester.acc2.Name, tester.acc1.Name, 15, "")))
	a.NoError(d.ProduceBlocks(1))

	a.Equal(balance0 - 40, tester.acc0.GetBalance().Value)
	a.Equal(balance1 + 25, tester.acc1.GetBalance().Value)
	a.Equal(balance2 + 15, tester.acc2.GetBalance().Value)
}

func (tester *TransferTester) tooMuch(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	balance0 := tester.acc0.GetBalance().Value
	balance1 := tester.acc1.GetBalance().Value

	a.NoError(tester.acc0.SendTrx(Transfer(tester.acc0.Name, tester.acc1.Name, balance0 + 1, "")))
	a.NoError(tester.acc0.SendTrx(Transfer(tester.acc0.Name, tester.acc1.Name, math.MaxUint64, "")))
	a.NoError(d.ProduceBlocks(1))

	a.Equal(balance0, tester.acc0.GetBalance().Value)
	a.Equal(balance1, tester.acc1.GetBalance().Value)
}

func (tester *TransferTester) toUnknown(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	balance0 := tester.acc0.GetBalance().Value
	a.NoError(tester.acc0.SendTrx(Transfer(tester.acc0.Name, "noexist1", 10, "")))
	a.NoError(tester.acc0.SendTrx(Transfer(tester.acc0.Name, "noexist2", 20, "")))
	a.NoError(tester.acc0.SendTrx(Transfer(tester.acc0.Name, "noexist3", 30, "")))
	a.NoError(d.ProduceBlocks(1))

	a.Equal(balance0, tester.acc0.GetBalance().Value)
}

func (tester *TransferTester) toSelf(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	balance0 := tester.acc0.GetBalance().Value
	a.NoError(tester.acc0.SendTrx(Transfer(tester.acc0.Name, tester.acc0.Name, 10, "")))
	a.NoError(d.ProduceBlocks(1))

	a.Equal(balance0, tester.acc0.GetBalance().Value)
}

func (tester *TransferTester) wrongSender(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	balance0 := tester.acc0.GetBalance().Value
	balance1 := tester.acc1.GetBalance().Value
	balance2 := tester.acc2.GetBalance().Value

	a.Error(tester.acc0.SendTrx(Transfer(tester.acc1.Name, tester.acc2.Name, 10, "")))
	a.Error(tester.acc1.SendTrx(Transfer(tester.acc0.Name, tester.acc1.Name, 10, "")))
	a.Error(tester.acc2.SendTrx(Transfer(tester.acc1.Name, tester.acc2.Name, 10, "")))
	a.NoError(d.ProduceBlocks(1))

	a.Equal(balance0, tester.acc0.GetBalance().Value)
	a.Equal(balance1, tester.acc1.GetBalance().Value)
	a.Equal(balance2, tester.acc2.GetBalance().Value)
}
