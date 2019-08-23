package economist

import (
	"github.com/coschain/contentos-go/common/constants"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

type DappTester struct {
	acc0,acc1,acc2,acc3,acc4 *DandelionAccount
}

func (tester *DappTester) Test(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")
	tester.acc3 = d.Account("actor3")
	tester.acc4 = d.Account("actor4")

	a := assert.New(t)
	registerBlockProducer(tester.acc4, t)

	const VEST = 1000

	a.NoError(tester.acc0.SendTrx(TransferToVest(tester.acc0.Name, tester.acc0.Name, VEST, "")))

	t.Run("normal self 100%", d.Test(tester.normal1))
	t.Run("normal self 50%", d.Test(tester.normal2))
	t.Run("normal other 100%", d.Test(tester.normal3))
	t.Run("normal self and other half-and-half", d.Test(tester.normal4))
	t.Run("normal three people", d.Test(tester.normal5))
	t.Run("normal reply dapp two people", d.Test(tester.normal6))
	t.Run("normal post and reply dapp", d.Test(tester.normal7))
}

func (tester *DappTester) normal1(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const POST = 1

	beneficiary := []map[string]int{{tester.acc0.Name: 10000}}
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, beneficiary)))

	acc0vest0 := d.Account(tester.acc0.Name).GetVest().Value

	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, POST)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - 2))

	// dapp reward
	dappWeight := StringToBigInt(d.Post(POST).GetWeightedVp())
	a.NotEqual(dappWeight.Uint64(), int64(0))

	globalDappReward := new(big.Int).SetUint64(d.GlobalProps().GetPoolDappRewards().Value)
	bigDappWvp := StringToBigInt(d.GlobalProps().GetWeightedVpsDapp())
	decayedDappWeight := bigDecay(bigDappWvp)
	exceptNextBlockDappWeightedVps := decayedDappWeight.Add(decayedDappWeight, dappWeight)
	nextBlockGlobalDappReward := globalDappReward.Add(globalDappReward, new(big.Int).SetUint64(perBlockDappReward(d)))
	dappReward := ProportionAlgorithm(dappWeight, exceptNextBlockDappWeightedVps, nextBlockGlobalDappReward)
	exceptGlobalClaimRewardAfterCashout := d.GlobalProps().GetClaimedDappRewards().Add(&prototype.Vest{Value: dappReward.Uint64()})
	exceptGlobalRewardAfterCashout := &prototype.Vest{ Value: new(big.Int).Sub(nextBlockGlobalDappReward, dappReward).Uint64()}
	// post reward
	postWeight := StringToBigInt(d.Post(POST).GetWeightedVp())
	a.NotEqual( postWeight.Int64(), int64(0) )

	globalPostReward := new(big.Int).SetUint64(d.GlobalProps().GetPoolPostRewards().Value)
	bigTotalPostWeight := StringToBigInt(d.GlobalProps().GetWeightedVpsPost())
	decayedPostWeight := bigDecay(bigTotalPostWeight)
	exceptNextBlockPostWeightedVps := decayedPostWeight.Add(decayedPostWeight, postWeight)
	nextBlockGlobalPostReward := globalPostReward.Add(globalPostReward, new(big.Int).SetUint64(perBlockPostReward(d)))
	postReward := ProportionAlgorithm(postWeight, exceptNextBlockPostWeightedVps, nextBlockGlobalPostReward)

	reward := new(big.Int).Add(postReward, dappReward)

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetWeightedVpsDapp(), exceptNextBlockDappWeightedVps.String())
	acc0vest1 := d.Account(tester.acc0.Name).GetVest().Value
	a.NotZero(reward.Uint64())
	a.Equal(reward.Uint64(), acc0vest1 - acc0vest0)
	a.Equal(d.Post(POST).GetDappRewards().Value, dappReward.Uint64())
	a.Equal(d.GlobalProps().GetClaimedDappRewards(), exceptGlobalClaimRewardAfterCashout)
	a.Equal(d.GlobalProps().GetPoolDappRewards(), exceptGlobalRewardAfterCashout)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *DappTester) normal2(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100
	const POST = 2

	beneficiary := []map[string]int{{tester.acc0.Name: 5000}}
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"2"}, beneficiary)))

	acc0vest0 := d.Account(tester.acc0.Name).GetVest().Value

	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, POST)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - 2))

	// dapp reward
	dappWeight := StringToBigInt(d.Post(POST).GetWeightedVp())
	dappWeight = ProportionAlgorithm(new(big.Int).SetUint64(5000), new(big.Int).SetUint64(constants.PERCENT), dappWeight)
	a.NotEqual(dappWeight.Uint64(), int64(0))

	globalDappReward := new(big.Int).SetUint64(d.GlobalProps().GetPoolDappRewards().Value)
	bigDappWvp := StringToBigInt(d.GlobalProps().GetWeightedVpsDapp())
	decayedDappWeight := bigDecay(bigDappWvp)
	exceptNextBlockDappWeightedVps := decayedDappWeight.Add(decayedDappWeight, dappWeight)
	nextBlockGlobalDappReward := globalDappReward.Add(globalDappReward, new(big.Int).SetUint64(perBlockDappReward(d)))
	dappReward := ProportionAlgorithm(dappWeight, exceptNextBlockDappWeightedVps, nextBlockGlobalDappReward)
	exceptGlobalClaimRewardAfterCashout := d.GlobalProps().GetClaimedDappRewards().Add(&prototype.Vest{Value: dappReward.Uint64()})
	exceptGlobalRewardAfterCashout := &prototype.Vest{ Value: new(big.Int).Sub(nextBlockGlobalDappReward, dappReward).Uint64()}
	// post reward
	postWeight := StringToBigInt(d.Post(POST).GetWeightedVp())
	a.NotEqual( postWeight.Int64(), int64(0) )

	globalPostReward := new(big.Int).SetUint64(d.GlobalProps().GetPoolPostRewards().Value)
	bigTotalPostWeight := StringToBigInt(d.GlobalProps().GetWeightedVpsPost())
	decayedPostWeight := bigDecay(bigTotalPostWeight)
	exceptNextBlockPostWeightedVps := decayedPostWeight.Add(decayedPostWeight, postWeight)
	nextBlockGlobalPostReward := globalPostReward.Add(globalPostReward, new(big.Int).SetUint64(perBlockPostReward(d)))
	postReward := ProportionAlgorithm(postWeight, exceptNextBlockPostWeightedVps, nextBlockGlobalPostReward)

	reward := new(big.Int).Add(postReward, dappReward)

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetWeightedVpsDapp(), exceptNextBlockDappWeightedVps.String())
	acc0vest1 := d.Account(tester.acc0.Name).GetVest().Value
	a.NotZero(reward.Uint64())
	a.Equal(reward.Uint64(), acc0vest1 - acc0vest0)
	a.Equal(d.Post(POST).GetDappRewards().Value, dappReward.Uint64())
	a.Equal(d.GlobalProps().GetClaimedDappRewards(), exceptGlobalClaimRewardAfterCashout)
	a.Equal(d.GlobalProps().GetPoolDappRewards(), exceptGlobalRewardAfterCashout)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *DappTester) normal3(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const POST = 3

	beneficiary := []map[string]int{{tester.acc2.Name: 10000}}
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"3"}, beneficiary)))

	acc0vest0 := d.Account(tester.acc2.Name).GetVest().Value

	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, POST)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - 2))

	// 100%
	dappWeight := StringToBigInt(d.Post(POST).GetWeightedVp())
	a.NotEqual(dappWeight.Uint64(), int64(0))

	globalDappReward := new(big.Int).SetUint64(d.GlobalProps().GetPoolDappRewards().Value)
	bigDappWvp := StringToBigInt(d.GlobalProps().GetWeightedVpsDapp())
	decayedDappWeight := bigDecay(bigDappWvp)
	exceptNextBlockDappWeightedVps := decayedDappWeight.Add(decayedDappWeight, dappWeight)
	nextBlockGlobalDappReward := globalDappReward.Add(globalDappReward, new(big.Int).SetUint64(perBlockDappReward(d)))
	dappReward := ProportionAlgorithm(dappWeight, exceptNextBlockDappWeightedVps, nextBlockGlobalDappReward)
	exceptGlobalClaimRewardAfterCashout := d.GlobalProps().GetClaimedDappRewards().Add(&prototype.Vest{Value: dappReward.Uint64()})
	exceptGlobalRewardAfterCashout := &prototype.Vest{ Value: new(big.Int).Sub(nextBlockGlobalDappReward, dappReward).Uint64()}

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetWeightedVpsDapp(), exceptNextBlockDappWeightedVps.String())
	acc0vest1 := d.Account(tester.acc2.Name).GetVest().Value
	a.NotZero(dappReward.Uint64())
	a.Equal(dappReward.Uint64(), acc0vest1 - acc0vest0)
	a.Equal(d.Post(POST).GetDappRewards().Value, dappReward.Uint64())
	a.Equal(d.GlobalProps().GetClaimedDappRewards(), exceptGlobalClaimRewardAfterCashout)
	a.Equal(d.GlobalProps().GetPoolDappRewards(), exceptGlobalRewardAfterCashout)
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *DappTester) normal4(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const POST = 4

	beneficiary := []map[string]int{{tester.acc0.Name: 5000}, {tester.acc2.Name: 5000}}
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, beneficiary)))

	acc0vest0 := d.Account(tester.acc0.Name).GetVest().Value
	acc1vest0 := d.Account(tester.acc2.Name).GetVest().Value

	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, POST)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - 2))

	// dapp reward
	dappWeight := StringToBigInt(d.Post(POST).GetWeightedVp())
	acc0DappWeight := ProportionAlgorithm(new(big.Int).SetUint64(5000), new(big.Int).SetUint64(constants.PERCENT), dappWeight)
	acc2DappWeight := ProportionAlgorithm(new(big.Int).SetUint64(5000), new(big.Int).SetUint64(constants.PERCENT), dappWeight)
	dappWeight = new(big.Int).Add(acc0DappWeight, acc2DappWeight)

	globalDappReward := new(big.Int).SetUint64(d.GlobalProps().GetPoolDappRewards().Value)
	bigDappWvp := StringToBigInt(d.GlobalProps().GetWeightedVpsDapp())
	decayedDappWeight := bigDecay(bigDappWvp)
	exceptNextBlockDappWeightedVps := decayedDappWeight.Add(decayedDappWeight, dappWeight)
	nextBlockGlobalDappReward := globalDappReward.Add(globalDappReward, new(big.Int).SetUint64(perBlockDappReward(d)))
	acc0DappReward := ProportionAlgorithm(acc0DappWeight, exceptNextBlockDappWeightedVps, nextBlockGlobalDappReward)
	acc2DappReward := ProportionAlgorithm(acc2DappWeight, exceptNextBlockDappWeightedVps, nextBlockGlobalDappReward)

	dappReward := new(big.Int).Add(acc0DappReward, acc2DappReward)
	exceptGlobalClaimRewardAfterCashout := d.GlobalProps().GetClaimedDappRewards().Add(&prototype.Vest{Value: dappReward.Uint64()})
	exceptGlobalRewardAfterCashout := &prototype.Vest{ Value: new(big.Int).Sub(nextBlockGlobalDappReward, dappReward).Uint64()}
	// post reward
	postWeight := StringToBigInt(d.Post(POST).GetWeightedVp())
	a.NotEqual( postWeight.Int64(), int64(0) )

	globalPostReward := new(big.Int).SetUint64(d.GlobalProps().GetPoolPostRewards().Value)
	bigTotalPostWeight := StringToBigInt(d.GlobalProps().GetWeightedVpsPost())
	decayedPostWeight := bigDecay(bigTotalPostWeight)
	exceptNextBlockPostWeightedVps := decayedPostWeight.Add(decayedPostWeight, postWeight)
	nextBlockGlobalPostReward := globalPostReward.Add(globalPostReward, new(big.Int).SetUint64(perBlockPostReward(d)))
	postReward := ProportionAlgorithm(postWeight, exceptNextBlockPostWeightedVps, nextBlockGlobalPostReward)

	acc0Reward := new(big.Int).Add(postReward, acc0DappReward)
	acc2Reward := acc2DappReward

	acc0acc2DappReward := new(big.Int).Add(acc0DappReward, acc0DappReward)

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetWeightedVpsDapp(), exceptNextBlockDappWeightedVps.String())

	acc0vest1 := d.Account(tester.acc0.Name).GetVest().Value
	acc1vest1 := d.Account(tester.acc2.Name).GetVest().Value

	a.NotZero(acc0Reward.Uint64())
	a.NotZero(acc2Reward.Uint64())
	a.Equal(acc0Reward.Uint64(), acc0vest1 - acc0vest0)
	a.Equal(acc2Reward.Uint64(), acc1vest1 - acc1vest0)
	a.Equal(d.Post(POST).GetDappRewards().Value, acc0acc2DappReward.Uint64())
	a.Equal(d.GlobalProps().GetClaimedDappRewards(), exceptGlobalClaimRewardAfterCashout)
	a.Equal(d.GlobalProps().GetPoolDappRewards(), exceptGlobalRewardAfterCashout)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *DappTester) normal5(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const POST = 5

	beneficiary := []map[string]int{{tester.acc0.Name: 5000}, {tester.acc2.Name: 2000}, {tester.acc3.Name: 2000}}
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"5"}, beneficiary)))

	acc0vest0 := d.Account(tester.acc0.Name).GetVest().Value
	acc1vest0 := d.Account(tester.acc2.Name).GetVest().Value
	acc2vest0 := d.Account(tester.acc3.Name).GetVest().Value

	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, POST)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - 2))

	// dapp reward
	dappWeight := StringToBigInt(d.Post(POST).GetWeightedVp())
	acc0DappWeight := ProportionAlgorithm(new(big.Int).SetUint64(5000), new(big.Int).SetUint64(constants.PERCENT), dappWeight)
	acc2DappWeight := ProportionAlgorithm(new(big.Int).SetUint64(2000), new(big.Int).SetUint64(constants.PERCENT), dappWeight)
	acc3DappWeight := ProportionAlgorithm(new(big.Int).SetUint64(2000), new(big.Int).SetUint64(constants.PERCENT), dappWeight)
	dappWeight = new(big.Int).Add(acc0DappWeight, acc2DappWeight)
	dappWeight.Add(dappWeight, acc3DappWeight)

	globalDappReward := new(big.Int).SetUint64(d.GlobalProps().GetPoolDappRewards().Value)
	bigDappWvp := StringToBigInt(d.GlobalProps().GetWeightedVpsDapp())
	decayedDappWeight := bigDecay(bigDappWvp)
	exceptNextBlockDappWeightedVps := decayedDappWeight.Add(decayedDappWeight, dappWeight)
	nextBlockGlobalDappReward := globalDappReward.Add(globalDappReward, new(big.Int).SetUint64(perBlockDappReward(d)))
	acc0DappReward := ProportionAlgorithm(acc0DappWeight, exceptNextBlockDappWeightedVps, nextBlockGlobalDappReward)
	acc2DappReward := ProportionAlgorithm(acc2DappWeight, exceptNextBlockDappWeightedVps, nextBlockGlobalDappReward)
	acc3DappReward := ProportionAlgorithm(acc3DappWeight, exceptNextBlockDappWeightedVps, nextBlockGlobalDappReward)

	dappReward := new(big.Int).Add(acc0DappReward, acc2DappReward)
	dappReward.Add(dappReward, acc3DappReward)
	exceptGlobalClaimRewardAfterCashout := d.GlobalProps().GetClaimedDappRewards().Add(&prototype.Vest{Value: dappReward.Uint64()})
	exceptGlobalRewardAfterCashout := &prototype.Vest{ Value: new(big.Int).Sub(nextBlockGlobalDappReward, dappReward).Uint64()}
	// post reward
	postWeight := StringToBigInt(d.Post(POST).GetWeightedVp())
	a.NotEqual( postWeight.Int64(), int64(0) )

	globalPostReward := new(big.Int).SetUint64(d.GlobalProps().GetPoolPostRewards().Value)
	bigTotalPostWeight := StringToBigInt(d.GlobalProps().GetWeightedVpsPost())
	decayedPostWeight := bigDecay(bigTotalPostWeight)
	exceptNextBlockPostWeightedVps := decayedPostWeight.Add(decayedPostWeight, postWeight)
	nextBlockGlobalPostReward := globalPostReward.Add(globalPostReward, new(big.Int).SetUint64(perBlockPostReward(d)))
	postReward := ProportionAlgorithm(postWeight, exceptNextBlockPostWeightedVps, nextBlockGlobalPostReward)

	acc0Reward := new(big.Int).Add(postReward, acc0DappReward)
	acc2Reward := acc2DappReward
	acc3Reward := acc3DappReward

	allAccDappReward := new(big.Int).Add(acc0DappReward, acc2DappReward)
	allAccDappReward.Add(allAccDappReward, acc3DappReward)

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetWeightedVpsDapp(), exceptNextBlockDappWeightedVps.String())

	acc0vest1 := d.Account(tester.acc0.Name).GetVest().Value
	acc1vest1 := d.Account(tester.acc2.Name).GetVest().Value
	acc2vest1 := d.Account(tester.acc3.Name).GetVest().Value

	a.NotZero(acc0Reward.Uint64())
	a.NotZero(acc2Reward.Uint64())
	a.Equal(acc0Reward.Uint64(), acc0vest1 - acc0vest0)
	a.Equal(acc2Reward.Uint64(), acc1vest1 - acc1vest0)
	a.Equal(acc3Reward.Uint64(), acc2vest1 - acc2vest0)
	a.Equal(d.Post(POST).GetDappRewards().Value, allAccDappReward.Uint64())
	a.Equal(d.GlobalProps().GetClaimedDappRewards(), exceptGlobalClaimRewardAfterCashout)
	a.Equal(d.GlobalProps().GetPoolDappRewards(), exceptGlobalRewardAfterCashout)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

// reply dapp
func (tester *DappTester) normal6(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const POST = 6
	const REPLY = 7

	beneficiary := []map[string]int{{tester.acc2.Name: 2000}, {tester.acc3.Name: 2000}}
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"5"}, nil)))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Reply(REPLY, POST,  tester.acc0.Name, "content",  beneficiary)))

	acc1vest0 := d.Account(tester.acc2.Name).GetVest().Value
	acc2vest0 := d.Account(tester.acc3.Name).GetVest().Value

	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, REPLY)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - 2))

	// dapp reward
	dappWeight := StringToBigInt(d.Post(REPLY).GetWeightedVp())
	acc2DappWeight := ProportionAlgorithm(new(big.Int).SetUint64(2000), new(big.Int).SetUint64(constants.PERCENT), dappWeight)
	acc3DappWeight := ProportionAlgorithm(new(big.Int).SetUint64(2000), new(big.Int).SetUint64(constants.PERCENT), dappWeight)
	// reply dapp equalize
	acc2DappWeightEqualize := ProportionAlgorithm(new(big.Int).SetUint64(constants.RewardRateReply), new(big.Int).SetUint64(constants.RewardRateAuthor), acc2DappWeight)
	acc3DappWeightEqualize := ProportionAlgorithm(new(big.Int).SetUint64(constants.RewardRateReply), new(big.Int).SetUint64(constants.RewardRateAuthor), acc3DappWeight)
	dappWeight = new(big.Int).Add(acc2DappWeightEqualize, acc3DappWeightEqualize)

	globalDappReward := new(big.Int).SetUint64(d.GlobalProps().GetPoolDappRewards().Value)
	bigDappWvp := StringToBigInt(d.GlobalProps().GetWeightedVpsDapp())
	decayedDappWeight := bigDecay(bigDappWvp)
	exceptNextBlockDappWeightedVps := decayedDappWeight.Add(decayedDappWeight, dappWeight)
	nextBlockGlobalDappReward := globalDappReward.Add(globalDappReward, new(big.Int).SetUint64(perBlockDappReward(d)))
	acc2DappReward := ProportionAlgorithm(acc2DappWeightEqualize, exceptNextBlockDappWeightedVps, nextBlockGlobalDappReward)
	acc3DappReward := ProportionAlgorithm(acc3DappWeightEqualize, exceptNextBlockDappWeightedVps, nextBlockGlobalDappReward)

	dappReward := new(big.Int).Add(acc2DappReward, acc3DappReward)

	exceptGlobalClaimRewardAfterCashout := d.GlobalProps().GetClaimedDappRewards().Add(&prototype.Vest{Value: dappReward.Uint64()})
	exceptGlobalRewardAfterCashout := &prototype.Vest{ Value: new(big.Int).Sub(nextBlockGlobalDappReward, dappReward).Uint64()}

	acc2Reward := acc2DappReward
	acc3Reward := acc3DappReward

	allAccDappReward := new(big.Int).Add(acc2DappReward, acc3DappReward)

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetWeightedVpsDapp(), exceptNextBlockDappWeightedVps.String())

	acc1vest1 := d.Account(tester.acc2.Name).GetVest().Value
	acc2vest1 := d.Account(tester.acc3.Name).GetVest().Value

	a.NotZero(acc2Reward.Uint64())
	a.Equal(acc2Reward.Uint64(), acc1vest1 - acc1vest0)
	a.Equal(acc3Reward.Uint64(), acc2vest1 - acc2vest0)
	a.Equal(d.Post(REPLY).GetDappRewards().Value, allAccDappReward.Uint64())
	a.Equal(d.GlobalProps().GetClaimedDappRewards(), exceptGlobalClaimRewardAfterCashout)
	a.Equal(d.GlobalProps().GetPoolDappRewards(), exceptGlobalRewardAfterCashout)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *DappTester) normal7(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const POST1 = 8
	const POST2 = 9
	const REPLY = 10

	postBeneficiary := []map[string]int{{tester.acc2.Name: 10000}}
	replyBeneficiary := []map[string]int{{tester.acc3.Name: 10000}}
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST1, tester.acc0.Name, "title", "content", []string{"5"}, nil)))
	a.NoError(tester.acc0.SendTrx(Post(POST2, tester.acc0.Name, "title", "content", []string{"6"}, postBeneficiary)))
	a.NoError(tester.acc1.SendTrx(Reply(REPLY, POST1,  tester.acc1.Name, "content",  replyBeneficiary)))
	a.NoError(d.ProduceBlocks(1))

	acc2vest0 := d.Account(tester.acc2.Name).GetVest().Value
	acc3vest0 := d.Account(tester.acc3.Name).GetVest().Value

	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, POST2)))
	a.NoError(tester.acc0.SendTrx(Vote(tester.acc0.Name, REPLY)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - 2))

	post1DappWeight := StringToBigInt(d.Post(POST2).GetWeightedVp())
	replyDappWeight := StringToBigInt(d.Post(REPLY).GetWeightedVp())
	// dapp reward
	acc2DappWeight := ProportionAlgorithm(new(big.Int).SetUint64(10000), new(big.Int).SetUint64(constants.PERCENT), post1DappWeight)
	acc3DappWeight := ProportionAlgorithm(new(big.Int).SetUint64(10000), new(big.Int).SetUint64(constants.PERCENT), replyDappWeight)
	// reply dapp equalize
	acc2DappWeightEqualize := acc2DappWeight
	acc3DappWeightEqualize := ProportionAlgorithm(new(big.Int).SetUint64(constants.RewardRateReply), new(big.Int).SetUint64(constants.RewardRateAuthor), acc3DappWeight)
	dappWeight := new(big.Int).Add(acc2DappWeightEqualize, acc3DappWeightEqualize)

	globalDappReward := new(big.Int).SetUint64(d.GlobalProps().GetPoolDappRewards().Value)
	bigDappWvp := StringToBigInt(d.GlobalProps().GetWeightedVpsDapp())
	decayedDappWeight := bigDecay(bigDappWvp)
	exceptNextBlockDappWeightedVps := decayedDappWeight.Add(decayedDappWeight, dappWeight)
	nextBlockGlobalDappReward := globalDappReward.Add(globalDappReward, new(big.Int).SetUint64(perBlockDappReward(d)))
	acc2DappReward := ProportionAlgorithm(acc2DappWeightEqualize, exceptNextBlockDappWeightedVps, nextBlockGlobalDappReward)
	acc3DappReward := ProportionAlgorithm(acc3DappWeightEqualize, exceptNextBlockDappWeightedVps, nextBlockGlobalDappReward)

	dappReward := new(big.Int).Add(acc2DappReward, acc3DappReward)

	exceptGlobalClaimRewardAfterCashout := d.GlobalProps().GetClaimedDappRewards().Add(&prototype.Vest{Value: dappReward.Uint64()})
	exceptGlobalRewardAfterCashout := &prototype.Vest{ Value: new(big.Int).Sub(nextBlockGlobalDappReward, dappReward).Uint64()}

	acc2Reward := acc2DappReward
	acc3Reward := acc3DappReward

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetWeightedVpsDapp(), exceptNextBlockDappWeightedVps.String())

	acc2vest1 := d.Account(tester.acc2.Name).GetVest().Value
	acc3vest1 := d.Account(tester.acc3.Name).GetVest().Value

	a.NotZero(acc2Reward.Uint64())
	a.Equal(acc2Reward.Uint64(), acc2vest1 - acc2vest0)
	a.Equal(acc3Reward.Uint64(), acc3vest1 - acc3vest0)
	a.Equal(d.GlobalProps().GetClaimedDappRewards(), exceptGlobalClaimRewardAfterCashout)
	a.Equal(d.GlobalProps().GetPoolDappRewards(), exceptGlobalRewardAfterCashout)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}