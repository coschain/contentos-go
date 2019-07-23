package op

import (
	"github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/dandelion/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

type TransferTester struct {
	acc0, acc1, acc2 *dandelion.DandelionAccount
}

func (tester *TransferTester) Test(t *testing.T, d *dandelion.Dandelion) {
	tester.acc0 = d.Account("testuser0")
	tester.acc1 = d.Account("testuser1")
	tester.acc2 = d.Account("testuser2")

	t.Run("normal", utils.TestWithDandelion(d, tester.normal))
	t.Run("too-much", utils.TestWithDandelion(d, tester.tooMuch))
	t.Run("to-unknown", utils.TestWithDandelion(d, tester.toUnknown))
	t.Run("to-self", utils.TestWithDandelion(d, tester.toSelf))
}

func (tester *TransferTester) normal(t *testing.T, d *dandelion.Dandelion) {
	a := assert.New(t)

	a.NoError(tester.acc0.SendTrx(utils.Transfer(tester.acc0.Name, tester.acc1.Name, 10, "")))
	a.NoError(tester.acc0.SendTrx(utils.Transfer(tester.acc0.Name, tester.acc2.Name, 30, "")))
	a.NoError(tester.acc2.SendTrx(utils.Transfer(tester.acc2.Name, tester.acc1.Name, 15, "")))
	a.NoError(d.ProduceBlocks(1))
}

func (tester *TransferTester) tooMuch(t *testing.T, d *dandelion.Dandelion) {

}

func (tester *TransferTester) toUnknown(t *testing.T, d *dandelion.Dandelion) {

}

func (tester *TransferTester) toSelf(t *testing.T, d *dandelion.Dandelion) {

}
