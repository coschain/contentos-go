package economist

import (
	"github.com/coschain/contentos-go/common/constants"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
	"github.com/stretchr/testify/assert"
	"testing"
)

type MintTester struct {
	acc0,acc1,acc2 *DandelionAccount
}

var mintProps = &prototype.ChainProperties{
	AccountCreationFee: prototype.NewCoin(1),
	MaximumBlockSize:   1024 * 1024,
	StaminaFree:        constants.DefaultStaminaFree,
	TpsExpected:        constants.DefaultTPSExpected,
	EpochDuration:      constants.InitEpochDuration,
	TopNAcquireFreeToken: constants.InitTopN,
	PerTicketPrice:     prototype.NewCoin(1000000),
	PerTicketWeight:    constants.PerTicketWeight,
}

func (tester *MintTester) BaseBudget(ith uint32) uint64 {
	if ith > 12 {
		return 0
	}
	var remain uint64 = 0
	if ith == 12 {
		remain = uint64(constants.TotalCurrency) * uint64(56) / 1000 / 100 * constants.BaseRate
	}
	return uint64(ith) * uint64(constants.TotalCurrency) * uint64(448) / 1000 / 100 * constants.BaseRate + remain
}

func (tester *MintTester) CalculateBudget(ith uint32) uint64 {
	return tester.BaseBudget(ith)
}

func (tester *MintTester) CalculatePerBlockBudget(annalBudget uint64) uint64 {
	return annalBudget / (86400 / constants.BlockInterval * 365)
}

func (tester *MintTester) Test(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	a := assert.New(t)
	a.NoError(tester.acc2.SendTrxAndProduceBlock(TransferToVest(tester.acc2.Name, tester.acc2.Name, constants.MinBpRegisterVest)))
	a.NoError(tester.acc2.SendTrxAndProduceBlock(BpRegister(tester.acc2.Name, "", "", tester.acc2.GetPubKey(), mintProps)))

	t.Run("normal", d.Test(tester.normal))
}

func (tester *MintTester) normal(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	ith := d.GlobalProps().GetIthYear()
	annualBudget := tester.CalculateBudget(ith)
	blockCurrency := tester.CalculatePerBlockBudget(annualBudget)

	const BLOCKS = 1000
	a.NoError(d.ProduceBlocks(BLOCKS))

	a.Equal(d.GlobalProps().GetPostRewards().Value, blockCurrency * constants.RewardRateCreator / constants.PERCENT * constants.RewardRateAuthor / constants.PERCENT * BLOCKS)
}


