package economist

import (
	"github.com/coschain/contentos-go/app/annual_mint"
	"github.com/coschain/contentos-go/common/constants"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/stretchr/testify/assert"
	"testing"
)

type MintTester struct {
	acc0,acc1,acc2 *DandelionAccount
}

func (tester *MintTester) Test(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	registerBlockProducer(tester.acc2, t)

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

	postRewards := d.GlobalProps().GetPoolPostRewards().Value
	replyRewards := d.GlobalProps().GetPoolReplyRewards().Value
	dappRewards := d.GlobalProps().GetPoolDappRewards().Value
	voteRewards := d.GlobalProps().GetPoolVoteRewards().Value
	bpVest := d.Account(tester.acc2.Name).GetVest().Value

	const BLOCKS = 1000
	a.NoError(d.ProduceBlocks(BLOCKS))

	creatorReward := blockCurrency * constants.RewardRateCreator / constants.PERCENT
	dappReward := blockCurrency * constants.RewardRateDapp / constants.PERCENT
	bpReward := blockCurrency - creatorReward - dappReward

	a.Equal(d.GlobalProps().GetPoolPostRewards().Value - postRewards, creatorReward * constants.RewardRateAuthor / constants.PERCENT * BLOCKS)
	a.Equal(d.GlobalProps().GetPoolReplyRewards().Value - replyRewards, creatorReward * constants.RewardRateReply / constants.PERCENT * BLOCKS)
	a.Equal(d.GlobalProps().GetPoolDappRewards().Value -  dappRewards, dappReward * BLOCKS)
	a.Equal(d.GlobalProps().GetPoolVoteRewards().Value -  voteRewards,
		(creatorReward - creatorReward * constants.RewardRateReply / constants.PERCENT - creatorReward * constants.RewardRateAuthor / constants.PERCENT) * BLOCKS)
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
	// the last block of the year mint less than ordinary block as the whole year should be mint annualBudget
	a.Equal(d.GlobalProps().GetIthYear(), ith + 1)
	a.Equal(d.GlobalProps().GetAnnualMinted().Value, d.GlobalProps().GetAnnualBudget().Value)
	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetIthYear(), ith + 1)
	a.Equal(d.GlobalProps().GetAnnualBudget().Value, annualBudget * 2)
}