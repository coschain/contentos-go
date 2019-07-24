package op

import (
	"github.com/coschain/contentos-go/common/constants"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
	"github.com/stretchr/testify/assert"
	"testing"
)

type BpTest struct {
	acc0, acc1, acc2 *DandelionAccount
}

func (tester *BpTest) Test(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	t.Run("normal", d.Test(tester.normal))
}

func (tester *BpTest) normal(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	props := &prototype.ChainProperties{
		AccountCreationFee: prototype.NewCoin(1),
		MaximumBlockSize:   1024 * 1024,
		StaminaFree:        constants.DefaultStaminaFree,
		TpsExpected:        constants.DefaultTPSExpected,
		EpochDuration:      constants.InitEpochDuration,
		TopNAcquireFreeToken: constants.InitTopN,
		PerTicketPrice:     prototype.NewCoin(1000000),
		PerTicketWeight:    constants.PerTicketWeight,
	}
	a.NoError(tester.acc0.SendTrx(BpRegister(tester.acc0.Name,"www.me.com","nothing",tester.acc0.GetOwner(),props)))
	a.NoError(d.ProduceBlocks(1))

	witWrap := d.Witness(tester.acc0.Name)
	a.True(witWrap.CheckExist())
}