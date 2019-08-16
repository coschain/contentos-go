package economist

import (
	"github.com/coschain/contentos-go/app/annual_mint"
	"github.com/coschain/contentos-go/common/constants"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

type VoteTester struct {
	acc0,acc1,acc2,acc3,acc4 *DandelionAccount
}

func perBlockVoteReward(d *Dandelion) uint64 {
	ith := d.GlobalProps().GetIthYear()
	annualBudget := annual_mint.CalculateBudget(ith)
	blockCurrency := annual_mint.CalculatePerBlockBudget(annualBudget)

	creatorReward := blockCurrency * constants.RewardRateCreator / constants.PERCENT

	postReward := creatorReward * constants.RewardRateAuthor / constants.PERCENT
	replyReward := creatorReward * constants.RewardRateReply / constants.PERCENT
	voterReward := creatorReward - postReward - replyReward
	return voterReward
}

func (tester *VoteTester) Test(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")
	tester.acc3 = d.Account("actor3")
	tester.acc4 = d.Account("actor4")

	a := assert.New(t)
	registerBlockProducer(tester.acc4, t)

	const VEST = 1000

	a.NoError(tester.acc0.SendTrx(TransferToVest(tester.acc0.Name, tester.acc0.Name, VEST, "")))
	a.NoError(tester.acc1.SendTrx(TransferToVest(tester.acc1.Name, tester.acc1.Name, VEST, "")))

	t.Run("normal", d.Test(tester.normal1))
	t.Run("normal cashout", d.Test(tester.normal2))
	t.Run("normal sequence cashout", d.Test(tester.normal3))
	t.Run("multi cashout", d.Test(tester.normal4))
}

func (tester *VoteTester) normal1(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100
	const POST = 1

	a.NoError(d.ProduceBlocks(BLOCKS))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, POST)))
	a.NoError(d.ProduceBlocks(constants.VoteCashOutDelayBlock - 1))
	vest0 := d.Account(tester.acc0.Name).GetVest().Value
	// to cashout
	a.NoError(d.ProduceBlocks(1))
	a.NotEqual(d.Account(tester.acc0.Name).GetVest().Value, vest0)

	vest1 := d.Account(tester.acc0.Name).GetVest().Value
	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.Account(tester.acc0.Name).GetVest().Value, vest1)
}

func (tester *VoteTester) normal2(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100
	const POST = 2

	a.NoError(d.ProduceBlocks(BLOCKS))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, POST)))
	a.NoError(d.ProduceBlocks(constants.VoteCashOutDelayBlock - 1))
	vest0 := d.Account(tester.acc1.Name).GetVest().Value

	postWeightedVp := StringToBigInt(d.Post(POST).GetWeightedVp())
	a.NotEqual(postWeightedVp.Int64(), int64(0))

	voteWeightedVp := StringToBigInt(d.Vote(tester.acc1.Name, POST).GetWeightedVp())
	weightedVp := new(big.Int).Mul(postWeightedVp, voteWeightedVp)
	decayedVoteWeight := bigDecay(StringToBigInt(d.GlobalProps().GetVoteWeightedVps()))
	totalVoteRewards := new(big.Int).SetUint64(d.GlobalProps().GetVoterRewards().Value)
	totalVoteRewards.Add(totalVoteRewards, new(big.Int).SetUint64(perBlockVoteReward(d)))
	totalVoteWeightedVp := decayedVoteWeight.Add(decayedVoteWeight, weightedVp)
	pr1 := new(big.Int).Mul(weightedVp, totalVoteRewards)
	exceptVoteReward := new(big.Int).Div(pr1, totalVoteWeightedVp)

	// to cashout
	a.NoError(d.ProduceBlocks(1))
	vest1 := d.Account(tester.acc1.Name).GetVest().Value
	a.Equal(d.GlobalProps().GetVoteWeightedVps(), totalVoteWeightedVp.String())
	a.Equal(vest1 - vest0, exceptVoteReward.Uint64())
}

func (tester *VoteTester) normal3(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100
	const POST = 3

	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	a.NoError(d.ProduceBlocks(BLOCKS))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Vote(tester.acc0.Name, POST)))
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, POST)))
	a.NoError(d.ProduceBlocks(constants.VoteCashOutDelayBlock - 1))
	vest0 := d.Account(tester.acc1.Name).GetVest().Value

	postWeightedVp := StringToBigInt(d.Post(POST).GetWeightedVp())
	a.NotEqual(postWeightedVp.Int64(), int64(0))

	voteWeightedVp := StringToBigInt(d.Vote(tester.acc1.Name, POST).GetWeightedVp())
	weightedVp := new(big.Int).Mul(postWeightedVp, voteWeightedVp)
	decayedVoteWeight := bigDecay(StringToBigInt(d.GlobalProps().GetVoteWeightedVps()))
	totalVoteRewards := new(big.Int).SetUint64(d.GlobalProps().GetVoterRewards().Value)
	totalVoteRewards.Add(totalVoteRewards, new(big.Int).SetUint64(perBlockVoteReward(d)))
	totalVoteWeightedVp := decayedVoteWeight.Add(decayedVoteWeight, weightedVp)
	pr1 := new(big.Int).Mul(weightedVp, totalVoteRewards)
	exceptVoteReward := new(big.Int).Div(pr1, totalVoteWeightedVp)

	// to cashout
	a.NoError(d.ProduceBlocks(1))
	vest1 := d.Account(tester.acc1.Name).GetVest().Value
	a.Equal(d.GlobalProps().GetVoteWeightedVps(), totalVoteWeightedVp.String())
	a.Equal(vest1 - vest0, exceptVoteReward.Uint64())
}

func (tester *VoteTester) normal4(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100
	const POST = 4

	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	a.NoError(d.ProduceBlocks(BLOCKS))
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, POST)))
	a.NoError(tester.acc2.SendTrx(Vote(tester.acc2.Name, POST)))
	a.NoError(d.ProduceBlocks(constants.VoteCashOutDelayBlock - 1))
	vest1 := d.Account(tester.acc1.Name).GetVest().Value
	vest2 := d.Account(tester.acc2.Name).GetVest().Value

	postWeightedVp := StringToBigInt(d.Post(POST).GetWeightedVp())
	a.NotEqual(postWeightedVp.Int64(), int64(0))

	vote1WeightedVp := StringToBigInt(d.Vote(tester.acc1.Name, POST).GetWeightedVp())
	vote2WeightedVp := StringToBigInt(d.Vote(tester.acc2.Name, POST).GetWeightedVp())
	weightedVp1 := new(big.Int).Mul(postWeightedVp, vote1WeightedVp)
	weightedVp2 := new(big.Int).Mul(postWeightedVp, vote2WeightedVp)
	weightedVp := new(big.Int).Add(weightedVp1, weightedVp2)
	decayedVoteWeight := bigDecay(StringToBigInt(d.GlobalProps().GetVoteWeightedVps()))
	totalVoteRewards := new(big.Int).SetUint64(d.GlobalProps().GetVoterRewards().Value)
	totalVoteRewards.Add(totalVoteRewards, new(big.Int).SetUint64(perBlockVoteReward(d)))
	totalVoteWeightedVp := decayedVoteWeight.Add(decayedVoteWeight, weightedVp)
	pr1 := new(big.Int).Mul(weightedVp, totalVoteRewards)
	exceptVoteReward := new(big.Int).Div(pr1, totalVoteWeightedVp)
	ratio1 := new(big.Int).Mul(weightedVp1, exceptVoteReward)
	reward1 := new(big.Int).Div(ratio1, weightedVp)
	ratio2 := new(big.Int).Mul(weightedVp2, exceptVoteReward)
	reward2 := new(big.Int).Div(ratio2, weightedVp)

	// to cashout
	a.NoError(d.ProduceBlocks(1))
	vest1n := d.Account(tester.acc1.Name).GetVest().Value
	vest2n := d.Account(tester.acc2.Name).GetVest().Value
	a.Equal(d.GlobalProps().GetVoteWeightedVps(), totalVoteWeightedVp.String())
	a.Equal(vest1n - vest1, reward1.Uint64())
	a.Equal(vest2n - vest2, reward2.Uint64())
}