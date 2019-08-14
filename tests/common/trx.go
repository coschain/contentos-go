package common

import (
	"github.com/coschain/contentos-go/common/constants"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

type TrxTester struct{}

func (tester *TrxTester) Test(t *testing.T, d *Dandelion) {
	t.Run("too_big", d.Test(tester.tooBig))
	t.Run("require_multi_signers", d.Test(tester.requireMultiSigners))
	t.Run("double_spent", d.Test(tester.doubleSpent))

}

func (tester *TrxTester) tooBig(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	// trxs with normal sizes should be accepted.
	a.NotNil(tester.transferWithMemo(d, ""))
	a.NotNil(tester.transferWithMemo(d, "your money"))

	// trxs larger than constants.MaxTransactionSize must be ignored.
	a.Nil(tester.transferWithMemo(d, strings.Repeat("A", constants.MaxTransactionSize)))
	a.Nil(tester.transferWithMemo(d, strings.Repeat("B", constants.MaxTransactionSize + 100)))
}

func (tester *TrxTester) doubleSpent(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	act1 := "actor1"
	act2 := "actor2"

	op := Transfer(act1, act2, 1, "double spent")

	prevBalance := d.Account(act1).GetBalance()

	trx, _, err := d.SendTrxEx2( d.GetAccountKey(act1), op )
	a.NoError(err)
	d.ProduceBlocks(1)
	a.Equal( prevBalance.Value - 1 , d.Account(act1).GetBalance().Value )

	// start double spent test
	for index := 0; index < constants.TrxMaxExpirationTime + 10 ; index++ {
		_, err = d.SendRawTrx(trx)
		d.ProduceBlocks(1)
		a.Error(err)
		a.Equal( prevBalance.Value - 1 , d.Account(act1).GetBalance().Value )
	}
}

func (tester *TrxTester) transferWithMemo(d *Dandelion, memo string) *prototype.TransactionReceiptWithInfo {
	return d.Account(constants.COSInitMiner).TrxReceipt(Transfer(constants.COSInitMiner, "actor0", 1, memo))
}

func (tester *TrxTester) requireMultiSigners(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	// normal case
	a.NotNil(d.Account(constants.COSInitMiner).TrxReceipt(
		Transfer(constants.COSInitMiner, "actor0", 2, ""),
	))
	// all operations in a trx must require the same signer.
	a.Nil(d.Account(constants.COSInitMiner).TrxReceipt(
		Transfer(constants.COSInitMiner, "actor0", 2, ""),
		Transfer("actor0", constants.COSInitMiner, 1, ""),
	))
}
