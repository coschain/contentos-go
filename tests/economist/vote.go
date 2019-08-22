package economist

import (
	"github.com/coschain/contentos-go/common/constants"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

type VoteTester struct {
	acc0,acc1,acc2,acc3,acc4 *DandelionAccount
}

func (tester *VoteTester) Test(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")
	tester.acc3 = d.Account("actor3")
	tester.acc4 = d.Account("actor4")

	registerBlockProducer(tester.acc4, t)

	const VEST = 1000
	SelfTransferToVesting([]*DandelionAccount{tester.acc0, tester.acc1}, VEST, t)

	t.Run("normal", d.Test(tester.normal1))
	t.Run("normal cashout", d.Test(tester.normal2))
	t.Run("normal sequence cashout", d.Test(tester.normal3))
	t.Run("multi cashout", d.Test(tester.normal4))
}

func (tester *VoteTester) normal1(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const POST = 1

	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	a.NoError(tester.acc1.SendTrxAndProduceBlock(Vote(tester.acc1.Name, POST)))
	a.NoError(d.ProduceBlocks(constants.VoteCashOutDelayBlock - 2))
	vest0 := d.Account(tester.acc1.Name).GetVest().Value
	// to cashout
	a.NoError(d.ProduceBlocks(1))
	a.NotEqual(d.Account(tester.acc1.Name).GetVest().Value, vest0)

	vest1 := d.Account(tester.acc1.Name).GetVest().Value
	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.Account(tester.acc1.Name).GetVest().Value, vest1)
}

func (tester *VoteTester) normal2(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const POST = 2

	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	a.NoError(tester.acc1.SendTrxAndProduceBlock(Vote(tester.acc1.Name, POST)))
	a.NoError(d.ProduceBlocks(constants.VoteCashOutDelayBlock - 2))
	vest0 := d.Account(tester.acc1.Name).GetVest().Value

	postWeightedVp := StringToBigInt(d.Post(POST).GetWeightedVp())
	a.NotEqual(postWeightedVp.Int64(), int64(0))

	voteWeightedVp := StringToBigInt(d.Vote(tester.acc1.Name, POST).GetWeightedVp())
	weightedVp := new(big.Int).Mul(postWeightedVp, voteWeightedVp)
	decayedVoteWeight := bigDecay(StringToBigInt(d.GlobalProps().GetWeightedVpsVote()))
	totalVoteRewards := new(big.Int).SetUint64(d.GlobalProps().GetPoolVoteRewards().Value)
	nextBlockGlobalVoteReward := new(big.Int).Add(totalVoteRewards, new(big.Int).SetUint64(perBlockVoteReward(d)))
	nextBlockGlobalVoteWeightedVp := new(big.Int).Add(decayedVoteWeight, weightedVp)
	exceptVoteReward := ProportionAlgorithm(weightedVp, nextBlockGlobalVoteWeightedVp, nextBlockGlobalVoteReward)

	exceptGlobalClaimRewardAfterCashout := d.GlobalProps().GetClaimedVoteRewards().Add(&prototype.Vest{Value: exceptVoteReward.Uint64()})
	exceptGlobalRewardAfterCashout := &prototype.Vest{ Value: new(big.Int).Sub(nextBlockGlobalVoteReward, exceptVoteReward).Uint64()}

	// to cashout
	a.NoError(d.ProduceBlocks(1))
	vest1 := d.Account(tester.acc1.Name).GetVest().Value
	a.Equal(d.GlobalProps().GetWeightedVpsVote(), nextBlockGlobalVoteWeightedVp.String())
	a.Equal(vest1 - vest0, exceptVoteReward.Uint64())
	a.Equal(d.GlobalProps().GetClaimedVoteRewards(), exceptGlobalClaimRewardAfterCashout)
	a.Equal(d.GlobalProps().GetPoolVoteRewards(), exceptGlobalRewardAfterCashout)
}

func (tester *VoteTester) normal3(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const POST = 3

	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	a.NoError(tester.acc2.SendTrxAndProduceBlock(Vote(tester.acc2.Name, POST)))
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, POST)))
	a.NoError(d.ProduceBlocks(constants.VoteCashOutDelayBlock - 2))

	acc1vest0 := d.Account(tester.acc1.Name).GetVest().Value
	acc2vest0 := d.Account(tester.acc2.Name).GetVest().Value

	postWeightedVp := StringToBigInt(d.Post(POST).GetWeightedVp())
	a.NotEqual(postWeightedVp.Int64(), int64(0))

	acc2VoteWeightedVp := StringToBigInt(d.Vote(tester.acc2.Name, POST).GetWeightedVp())
	acc2WeightedVp := new(big.Int).Mul(postWeightedVp, acc2VoteWeightedVp)
	decayedVoteWeight := bigDecay(StringToBigInt(d.GlobalProps().GetWeightedVpsVote()))
	totalVoteRewards := new(big.Int).SetUint64(d.GlobalProps().GetPoolVoteRewards().Value)
	nextBlockGlobalVoteReward := new(big.Int).Add(totalVoteRewards, new(big.Int).SetUint64(perBlockVoteReward(d)))
	nextBlockGlobalVoteWeightedVp := new(big.Int).Add(decayedVoteWeight, acc2WeightedVp)
	exceptVoteReward := ProportionAlgorithm(acc2WeightedVp, nextBlockGlobalVoteWeightedVp, nextBlockGlobalVoteReward)

	exceptGlobalClaimRewardAfterCashout := d.GlobalProps().GetClaimedVoteRewards().Add(&prototype.Vest{Value: exceptVoteReward.Uint64()})
	exceptGlobalRewardAfterCashout := &prototype.Vest{ Value: new(big.Int).Sub(nextBlockGlobalVoteReward, exceptVoteReward).Uint64()}

	// to cashout
	a.NoError(d.ProduceBlocks(1))
	acc2vest1 := d.Account(tester.acc2.Name).GetVest().Value
	a.Equal(d.GlobalProps().GetWeightedVpsVote(), nextBlockGlobalVoteWeightedVp.String())
	a.Equal(acc2vest1 - acc2vest0, exceptVoteReward.Uint64())
	a.Equal(d.GlobalProps().GetClaimedVoteRewards(), exceptGlobalClaimRewardAfterCashout)
	a.Equal(d.GlobalProps().GetPoolVoteRewards(), exceptGlobalRewardAfterCashout)

	acc1VoteWeightedVp := StringToBigInt(d.Vote(tester.acc1.Name, POST).GetWeightedVp())
	acc1WeightedVp := new(big.Int).Mul(postWeightedVp, acc1VoteWeightedVp)
	decayedVoteWeight = bigDecay(nextBlockGlobalVoteWeightedVp)
	// subtract rewards which had been cashout to acc2
	nextBlockGlobalVoteReward.Sub(nextBlockGlobalVoteReward, exceptVoteReward)
	nextBlockGlobalVoteReward.Add(nextBlockGlobalVoteReward, new(big.Int).SetUint64(perBlockVoteReward(d)))
	nextBlockGlobalVoteWeightedVp = new(big.Int).Add(decayedVoteWeight, acc1WeightedVp)
	exceptVoteReward = ProportionAlgorithm(acc1WeightedVp, nextBlockGlobalVoteWeightedVp, nextBlockGlobalVoteReward)

	exceptGlobalClaimRewardAfterCashout.Add(&prototype.Vest{Value: exceptVoteReward.Uint64()})
	exceptGlobalRewardAfterCashout = &prototype.Vest{ Value: new(big.Int).Sub(nextBlockGlobalVoteReward, exceptVoteReward).Uint64()}

	a.NoError(d.ProduceBlocks(1))
	acc1vest1 := d.Account(tester.acc1.Name).GetVest().Value
	a.Equal(d.GlobalProps().GetWeightedVpsVote(), nextBlockGlobalVoteWeightedVp.String())
	a.Equal(acc1vest1 - acc1vest0, exceptVoteReward.Uint64())
	a.Equal(d.GlobalProps().GetClaimedVoteRewards(), exceptGlobalClaimRewardAfterCashout)
	a.Equal(d.GlobalProps().GetPoolVoteRewards(), exceptGlobalRewardAfterCashout)
}

func (tester *VoteTester) normal4(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const POST = 4

	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, POST)))
	a.NoError(tester.acc2.SendTrx(Vote(tester.acc2.Name, POST)))
	a.NoError(d.ProduceBlocks(1))
	a.NoError(d.ProduceBlocks(constants.VoteCashOutDelayBlock - 2))

	vest1 := d.Account(tester.acc1.Name).GetVest().Value
	vest2 := d.Account(tester.acc2.Name).GetVest().Value

	postWeightedVp := StringToBigInt(d.Post(POST).GetWeightedVp())
	a.NotEqual(postWeightedVp.Int64(), int64(0))

	vote1WeightedVp := StringToBigInt(d.Vote(tester.acc1.Name, POST).GetWeightedVp())
	vote2WeightedVp := StringToBigInt(d.Vote(tester.acc2.Name, POST).GetWeightedVp())

	weightedVp1 := new(big.Int).Mul(postWeightedVp, vote1WeightedVp)
	weightedVp2 := new(big.Int).Mul(postWeightedVp, vote2WeightedVp)
	weightedVp := new(big.Int).Add(weightedVp1, weightedVp2)
	decayedVoteWeight := bigDecay(StringToBigInt(d.GlobalProps().GetWeightedVpsVote()))
	currentGlobalVoteReward := new(big.Int).SetUint64(d.GlobalProps().GetPoolVoteRewards().Value)
	nextBlockGlobalVoteReward := new(big.Int).Add(currentGlobalVoteReward, new(big.Int).SetUint64(perBlockVoteReward(d)))
	nextBlockGlobalVoteWeightedVp := decayedVoteWeight.Add(decayedVoteWeight, weightedVp)

	reward1 := ProportionAlgorithm(weightedVp1, nextBlockGlobalVoteWeightedVp, nextBlockGlobalVoteReward)
	reward2 := ProportionAlgorithm(weightedVp2, nextBlockGlobalVoteWeightedVp, nextBlockGlobalVoteReward)

	reward := new(big.Int).Add(reward1, reward2)

	exceptGlobalClaimRewardAfterCashout := d.GlobalProps().GetClaimedVoteRewards().Add(&prototype.Vest{Value: reward.Uint64()})
	exceptGlobalRewardAfterCashout := &prototype.Vest{ Value: new(big.Int).Sub(nextBlockGlobalVoteReward, reward).Uint64()}

	// to cashout
	a.NoError(d.ProduceBlocks(1))
	vest1n := d.Account(tester.acc1.Name).GetVest().Value
	vest2n := d.Account(tester.acc2.Name).GetVest().Value
	a.Equal(d.GlobalProps().GetWeightedVpsVote(), nextBlockGlobalVoteWeightedVp.String())
	a.Equal(vest1n - vest1, reward1.Uint64())
	a.Equal(vest2n - vest2, reward2.Uint64())
	a.Equal(d.GlobalProps().GetClaimedVoteRewards(), exceptGlobalClaimRewardAfterCashout)
	a.Equal(d.GlobalProps().GetPoolVoteRewards(), exceptGlobalRewardAfterCashout)
}