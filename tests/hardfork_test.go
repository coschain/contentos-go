package tests

import (
	"errors"
	"testing"

	. "github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
	"github.com/stretchr/testify/assert"
)

func checkError(r *prototype.TransactionReceiptWithInfo) error {
	if r == nil {
		return errors.New("receipt is nil")
	}
	if r.Status != prototype.StatusSuccess {
		return errors.New(r.ErrorInfo)
	}
	return nil
}

func TestHardfork(t *testing.T) {
	t.Run("follow", NewDandelionTest(new(NewOperationTester).Test, 2))
}

type NewOperationTester struct {
	acc0, acc1 *DandelionAccount
}

func (tester *NewOperationTester) Test(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")

	t.Run("follow", d.Test(tester.follow))
}

// the follow evaluator doesn't apply anything
func (tester *NewOperationTester) follow(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	d.ProduceBlocks(7) // head block 9
	a.Panics(func() {
		tester.acc0.TrxReceipt(Follow(tester.acc0.Name, tester.acc1.Name, false))
	}, "op not exist")

	d.ProduceTempBlock() // head block 10
	a.NoError(checkError(tester.acc0.TrxReceipt(Follow(tester.acc0.Name, tester.acc1.Name, false))))

	d.PopBlock(10)
	a.Panics(func() {
		tester.acc0.TrxReceipt(Follow(tester.acc0.Name, tester.acc1.Name, false))
	}, "op not exist")

	d.ProduceBlocks(1)
	a.NoError(checkError(tester.acc0.TrxReceipt(Follow(tester.acc0.Name, tester.acc1.Name, false))))
}
