package economist

import (
	"github.com/coschain/contentos-go/app/annual_mint"
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

func (tester *MintTester) Test(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	a := assert.New(t)
	a.NoError(tester.acc2.SendTrxAndProduceBlock(TransferToVest(tester.acc2.Name, tester.acc2.Name, constants.MinBpRegisterVest)))
	a.NoError(tester.acc2.SendTrxAndProduceBlock(BpRegister(tester.acc2.Name, "", "", tester.acc2.GetPubKey(), mintProps)))

	t.Run("normal", d.Test(tester.normal))
	t.Run("year", d.Test(tester.yearSwitch))
}

func (tester *MintTester) normal(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const TOSHUFFLEBLOCK = 5
	a.NoError(d.ProduceBlocks(TOSHUFFLEBLOCK))

	ith := d.GlobalProps().GetIthYear()
	annualBudget := annual_mint.CalculateBudget(ith)
	blockCurrency := annual_mint.CalculatePerBlockBudget(annualBudget)

	postRewards := d.GlobalProps().GetPostRewards().Value
	replyRewards := d.GlobalProps().GetReplyRewards().Value
	replyDappRewards := d.GlobalProps().GetReplyDappRewards().Value
	postDappRewards := d.GlobalProps().GetPostDappRewards().Value
	bpVest := d.Account(tester.acc2.Name).GetVest().Value

	const BLOCKS = 1000
	a.NoError(d.ProduceBlocks(BLOCKS))

	creatorReward := blockCurrency * constants.RewardRateCreator / constants.PERCENT
	dappReward := blockCurrency * constants.RewardRateDapp / constants.PERCENT
	bpReward := blockCurrency - creatorReward - dappReward

	a.Equal(d.GlobalProps().GetPostRewards().Value - postRewards, creatorReward * constants.RewardRateAuthor / constants.PERCENT * BLOCKS)
	a.Equal(d.GlobalProps().GetReplyRewards().Value - replyRewards, creatorReward * constants.RewardRateReply / constants.PERCENT * BLOCKS)
	a.Equal(d.GlobalProps().GetReplyDappRewards().Value - replyDappRewards, dappReward * constants.RewardRateReply / constants.PERCENT * BLOCKS)
	a.Equal(d.GlobalProps().GetPostDappRewards().Value - postDappRewards, (dappReward - dappReward * constants.RewardRateReply / constants.PERCENT) * BLOCKS)
	a.Equal(d.Account(tester.acc2.Name).GetVest().Value - bpVest, bpReward * BLOCKS)
}

func (tester *MintTester) yearSwitch(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	ith := d.GlobalProps().GetIthYear()
	annualBudget := annual_mint.CalculateBudget(ith)
	currentMinted := d.GlobalProps().GetAnnualMinted().Value
	currentBlockNum := d.GlobalProps().GetHeadBlockNumber()
	blockCurrency := annual_mint.CalculatePerBlockBudget(annualBudget)
	a.Equal(annualBudget, d.GlobalProps().GetAnnualBudget().Value)
	for {
		props := d.GlobalProps()
		if props.GetAnnualMinted().Value + blockCurrency > props.GetAnnualBudget().Value {
			break
		}
		a.NoError(d.ProduceBlocks(1))
	}
	props := d.GlobalProps()
	// the early blocks may not mint so it would not be 8640
	// stop before the switch block
	a.Equal(props.HeadBlockNumber, (annualBudget - currentMinted) / blockCurrency + currentBlockNum)
	a.Equal(props.GetIthYear(), ith)
	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetIthYear(), ith + 1)
	a.Equal(d.GlobalProps().GetAnnualMinted().Value, d.GlobalProps().GetAnnualBudget().Value)
	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetIthYear(), ith + 1)
	a.Equal(d.GlobalProps().GetAnnualBudget().Value, annualBudget * 2)
}


