package economist

import (
	"github.com/coschain/contentos-go/app"
	"github.com/coschain/contentos-go/common/constants"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
	"github.com/stretchr/testify/assert"
	"math"
	"math/big"
	"testing"
)

type PostTester struct {
	acc0,acc1,acc2 *DandelionAccount
}

func (tester *PostTester) Test1(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	registerBlockProducer(tester.acc2, t)

	const VEST = 1000
	SelfTransferToVesting([]*DandelionAccount{tester.acc0, tester.acc1}, VEST, t)

	t.Run("normal", d.Test(tester.normal))
}

func (tester *PostTester) Test2(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	registerBlockProducer(tester.acc2, t)

	const VEST = 1000
	SelfTransferToVesting([]*DandelionAccount{tester.acc0, tester.acc1}, VEST, t)

	t.Run("cashout", d.Test(tester.cashout))
	t.Run("cashout after other cashout", d.Test(tester.cashoutAfterOtherCashout))
	t.Run("mul cashout", d.Test(tester.multiCashout))
}

func (tester *PostTester) Test3(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	registerBlockProducer(tester.acc2, t)

	const VEST = 1000
	SelfTransferToVesting([]*DandelionAccount{tester.acc0, tester.acc1}, VEST, t)

	t.Run("huge global vp", d.Test(tester.hugeGlobalVp))
	//t.Run("zero global vp", d.Test(tester.zeroGlobalVp))
}

func (tester *PostTester) Test4(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	registerBlockProducer(tester.acc2, t)
	const VEST = 1000
	SelfTransferToVesting([]*DandelionAccount{tester.acc0, tester.acc1}, VEST, t)

	t.Run("with ticket", d.Test(tester.withTicket))
}


func (tester *PostTester) normal(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const POST = 1

	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	post1Block := d.GlobalProps().GetHeadBlockNumber() - 1
	post1Cashout := post1Block + constants.PostCashOutDelayBlock
	a.Equal(post1Cashout, d.Post(POST).GetCashoutBlockNum())
	// waiting for vp charge
	// next block post will be cashout
	vest0 := d.Account(tester.acc0.Name).GetVest().Value
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, POST)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - 2))
	a.Equal(d.Account(tester.acc0.Name).GetVest().Value, vest0)

	// to cashout
	a.NoError(d.ProduceBlocks(1))
	a.NotEqual(d.Account(tester.acc0.Name).GetVest().Value, vest0)

	vest1 := d.Account(tester.acc0.Name).GetVest().Value
	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.Account(tester.acc0.Name).GetVest().Value, vest1)
	a.Equal(d.Post(POST).GetCashoutBlockNum(), app.CashoutCompleted)
}

func (tester *PostTester) cashout(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const POST = 1

	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	post1Block := d.GlobalProps().GetHeadBlockNumber() - 1
	post1Cashout := post1Block + constants.PostCashOutDelayBlock
	a.Equal(post1Cashout, d.Post(POST).GetCashoutBlockNum())

	vest0 := d.Account(tester.acc0.Name).GetVest().Value
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, 1)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - 2))

	postWeight := StringToBigInt(d.Post(POST).GetWeightedVp())
	a.NotEqual( postWeight.Int64(), int64(0) )
	globalPostReward := new(big.Int).SetUint64(d.GlobalProps().GetPoolPostRewards().Value)
	bigTotalPostWeight := StringToBigInt(d.GlobalProps().GetWeightedVpsPost())
	decayedPostWeight := bigDecay(bigTotalPostWeight)
	exceptNextBlockPostWeightedVps := decayedPostWeight.Add(decayedPostWeight, postWeight)
	bigGlobalPostReward := globalPostReward.Add(globalPostReward, new(big.Int).SetUint64(perBlockPostReward(d)))
	reward := ProportionAlgorithm(postWeight, exceptNextBlockPostWeightedVps, bigGlobalPostReward)
	exceptGlobalRewardAfterCashout := new(big.Int).Sub(bigGlobalPostReward, reward)

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetWeightedVpsPost(), exceptNextBlockPostWeightedVps.String())
	vest1 := d.Account(tester.acc0.Name).GetVest().Value
	realReward := vest1 - vest0
	a.Equal(reward.Uint64(), realReward)
	a.Equal(d.Post(POST).GetRewards().Value, realReward)
	a.Equal(d.Post(POST).GetCashoutBlockNum(), app.CashoutCompleted)
	a.Equal(d.GlobalProps().GetClaimedPostRewards(), &prototype.Vest{Value: reward.Uint64()})
	a.Equal(d.GlobalProps().GetPoolPostRewards(), &prototype.Vest{Value: exceptGlobalRewardAfterCashout.Uint64()})
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *PostTester) cashoutAfterOtherCashout(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const POST = 2

	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"2"}, nil)))

	vest0 := d.Account(tester.acc0.Name).GetVest().Value
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, POST)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - 2 ))

	postWeight := StringToBigInt(d.Post(POST).GetWeightedVp())
	a.NotEqual(postWeight.Int64(), int64(0))
	globalPostReward := new(big.Int).SetUint64(d.GlobalProps().GetPoolPostRewards().Value)
	bigTotalPostWeight := StringToBigInt(d.GlobalProps().GetWeightedVpsPost())
	decayedPostWeight := bigDecay(bigTotalPostWeight)
	exceptNextBlockPostWeightedVps := decayedPostWeight.Add(decayedPostWeight, postWeight)
	bigGlobalPostReward := globalPostReward.Add(globalPostReward, new(big.Int).SetUint64(perBlockPostReward(d)))
	reward := ProportionAlgorithm(postWeight, exceptNextBlockPostWeightedVps, bigGlobalPostReward)
	exceptGlobalClaimRewardAfterCashout := d.GlobalProps().GetClaimedPostRewards().Add(&prototype.Vest{Value: reward.Uint64()})
	//exceptGlobalRewardAfterCashout := new(big.Int).Sub(bigGlobalPostReward, reward)
	exceptGlobalRewardAfterCashout := &prototype.Vest{ Value: new(big.Int).Sub(bigGlobalPostReward, reward).Uint64()}

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetWeightedVpsPost(), exceptNextBlockPostWeightedVps.String())
	vest1 := d.Account(tester.acc0.Name).GetVest().Value
	realReward := vest1 - vest0
	a.Equal(reward.Uint64(), realReward)
	a.Equal(d.GlobalProps().GetClaimedPostRewards(), exceptGlobalClaimRewardAfterCashout)
	a.Equal(d.GlobalProps().GetPoolPostRewards(), exceptGlobalRewardAfterCashout)
	a.Equal(d.Post(POST).GetRewards().Value, realReward)
	a.Equal(d.Post(POST).GetCashoutBlockNum(), app.CashoutCompleted)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *PostTester) multiCashout(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	const POST1 = 3
	const POST2 = 4
	const BLOCK = 100

	a.NoError(tester.acc0.SendTrx(Post(POST1, tester.acc0.Name, "title", "content", []string{"3"}, nil)))
	a.NoError(tester.acc1.SendTrx(Post(POST2, tester.acc1.Name, "title", "content", []string{"4"}, nil)))
	a.NoError(d.ProduceBlocks(1))
	// prevent vote cashout
	a.NoError(d.ProduceBlocks(BLOCK))

	vestold3 := d.Account(tester.acc0.Name).GetVest().Value
	vestold4 := d.Account(tester.acc1.Name).GetVest().Value
	a.NoError(tester.acc0.SendTrx(Vote(tester.acc0.Name, POST2)))
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, POST1)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - BLOCK - 2))

	// convert to uint64 to make test easier
	// the mul result less than uint64.MAX
	post3Weight :=  StringToBigInt(d.Post(POST1).GetWeightedVp())
	post4Weight :=  StringToBigInt(d.Post(POST2).GetWeightedVp())

	a.NotEqual(post3Weight.Int64(), int64(0))
	a.NotEqual(post4Weight.Int64(), int64(0))

	postsWeight := new(big.Int).Add(post3Weight, post4Weight)

	globalPostReward := new(big.Int).SetUint64(d.GlobalProps().GetPoolPostRewards().Value)
	bigTotalPostWeight := StringToBigInt(d.GlobalProps().GetWeightedVpsPost())

	decayedPostWeight := bigDecay(bigTotalPostWeight)
	exceptNextBlockPostWeightedVps := decayedPostWeight.Add(decayedPostWeight, postsWeight)
	bigGlobalPostReward := globalPostReward.Add(globalPostReward, new(big.Int).SetUint64(perBlockPostReward(d)))
	// new-style
	post3Reward := ProportionAlgorithm(post3Weight, exceptNextBlockPostWeightedVps, bigGlobalPostReward)
	post4Reward := ProportionAlgorithm(post4Weight, exceptNextBlockPostWeightedVps, bigGlobalPostReward)
	postsReward := new(big.Int).Add(post3Reward, post4Reward)
	exceptGlobalClaimRewardAfterCashout := d.GlobalProps().ClaimedPostRewards.Add(&prototype.Vest{Value: postsReward.Uint64()})
	exceptGlobalRewardAfterCashout := &prototype.Vest{ Value: new(big.Int).Sub(bigGlobalPostReward, postsReward).Uint64()}

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetWeightedVpsPost(), exceptNextBlockPostWeightedVps.String())
	vestnew3 := d.Account(tester.acc0.Name).GetVest().Value
	vestnew4 := d.Account(tester.acc1.Name).GetVest().Value
	real3Reward := vestnew3 - vestold3
	real4Reward := vestnew4 - vestold4
	a.Equal(post3Reward.Uint64(), real3Reward)
	a.Equal(post4Reward.Uint64(), real4Reward)
	a.Equal(d.GlobalProps().GetClaimedPostRewards(), exceptGlobalClaimRewardAfterCashout)
	a.Equal(d.GlobalProps().GetPoolPostRewards(), exceptGlobalRewardAfterCashout)
	a.Equal(d.Post(POST1).GetRewards().Value, real3Reward)
	a.Equal(d.Post(POST2).GetRewards().Value, real4Reward)
	a.Equal(d.Post(POST1).GetCashoutBlockNum(), app.CashoutCompleted)
	a.Equal(d.Post(POST2).GetCashoutBlockNum(), app.CashoutCompleted)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *PostTester) hugeGlobalVp(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100
	const POST = 1

	_ = d.ModifyProps(func(props *prototype.DynamicProperties) {
		maxUint64 := new(big.Int).SetUint64(math.MaxUint64)
		factor := new(big.Int).SetUint64(10)
		postWeightedVp := new(big.Int).Mul(maxUint64, factor)
		props.WeightedVpsPost = postWeightedVp.String()
	})
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, nil)))

	vest0 := d.Account(tester.acc0.Name).GetVest().Value
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, POST)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - 2))

	postWeight := StringToBigInt(d.Post(POST).GetWeightedVp())
	a.NotEqual(postWeight.Int64(), int64(0))
	globalPostReward := new(big.Int).SetUint64(d.GlobalProps().GetPoolPostRewards().Value)
	bigTotalPostWeight := StringToBigInt(d.GlobalProps().GetWeightedVpsPost())
	decayedPostWeight := bigDecay(bigTotalPostWeight)
	exceptNextBlockPostWeightedVps := decayedPostWeight.Add(decayedPostWeight, postWeight)
	bigGlobalPostReward := globalPostReward.Add(globalPostReward, new(big.Int).SetUint64(perBlockPostReward(d)))
	reward := ProportionAlgorithm(postWeight, exceptNextBlockPostWeightedVps, bigGlobalPostReward)
	exceptGlobalClaimRewardAfterCashout := d.GlobalProps().GetClaimedPostRewards().Add(&prototype.Vest{Value: reward.Uint64()})
	//exceptGlobalRewardAfterCashout := new(big.Int).Sub(bigGlobalPostReward, reward)
	exceptGlobalRewardAfterCashout := &prototype.Vest{ Value: new(big.Int).Sub(bigGlobalPostReward, reward).Uint64()}

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetWeightedVpsPost(), exceptNextBlockPostWeightedVps.String() )
	vest1 := d.Account(tester.acc0.Name).GetVest().Value
	realReward := vest1 - vest0
	a.Equal(reward.Uint64(), realReward)
	a.Equal(d.GlobalProps().GetClaimedPostRewards(), exceptGlobalClaimRewardAfterCashout)
	a.Equal(d.GlobalProps().GetPoolPostRewards(), exceptGlobalRewardAfterCashout)
	a.Equal(d.Post(POST).GetRewards().Value, realReward)
	a.Equal(d.Post(POST).GetCashoutBlockNum(), app.CashoutCompleted)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *PostTester) zeroGlobalVp(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const POST = 2

	_ = d.ModifyProps(func(props *prototype.DynamicProperties) {
		postWeightedVp := new(big.Int).SetUint64(0)
		props.WeightedVpsPost = postWeightedVp.String()
	})
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"2"}, nil)))

	vest0 := d.Account(tester.acc0.Name).GetVest().Value
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, POST)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - 2))

	postWeight := StringToBigInt(d.Post(POST).GetWeightedVp())
	a.NotEqual(postWeight.Int64(), int64(0))

	globalPostReward := new(big.Int).SetUint64(d.GlobalProps().GetPoolPostRewards().Value)
	bigTotalPostWeight := StringToBigInt(d.GlobalProps().GetWeightedVpsPost())
	decayedPostWeight := bigDecay(bigTotalPostWeight)
	exceptNextBlockPostWeightedVps := decayedPostWeight.Add(decayedPostWeight, postWeight)
	bigGlobalPostReward := globalPostReward.Add(globalPostReward, new(big.Int).SetUint64(perBlockPostReward(d)))
	reward := ProportionAlgorithm(postWeight, exceptNextBlockPostWeightedVps, bigGlobalPostReward)
	exceptGlobalClaimRewardAfterCashout := d.GlobalProps().GetClaimedPostRewards().Add(&prototype.Vest{Value: reward.Uint64()})
	//exceptGlobalRewardAfterCashout := new(big.Int).Sub(bigGlobalPostReward, reward)
	exceptGlobalRewardAfterCashout := &prototype.Vest{ Value: new(big.Int).Sub(bigGlobalPostReward, reward).Uint64()}

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetWeightedVpsPost(), exceptNextBlockPostWeightedVps.String() )
	vest1 := d.Account(tester.acc0.Name).GetVest().Value
	realReward := vest1 - vest0
	a.Equal(reward.Uint64(), realReward)
	a.Equal(d.Post(POST).GetRewards().Value, realReward)
	a.Equal(d.Post(POST).GetCashoutBlockNum(), app.CashoutCompleted)
	a.Equal(d.GlobalProps().GetClaimedPostRewards(), exceptGlobalClaimRewardAfterCashout)
	a.Equal(d.GlobalProps().GetPoolPostRewards(), exceptGlobalRewardAfterCashout)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *PostTester) withTicket(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const POST = 1

	const VEST = 10 * constants.COSTokenDecimals

	a.NoError(tester.acc0.SendTrxAndProduceBlock(TransferToVest(tester.acc0.Name, tester.acc0.Name, VEST, "")))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(AcquireTicket(tester.acc0.Name, 1)))

	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	a.NoError(tester.acc0.SendTrx(VoteByTicket(tester.acc0.Name, POST, 1)))

	vest0 := d.Account(tester.acc0.Name).GetVest().Value
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, POST)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - 2))

	postWeight := StringToBigInt(d.Post(POST).GetWeightedVp())
	a.NotEqual(postWeight.Int64(), int64(0))

	postWeight = new(big.Int).Add(postWeight, new(big.Int).SetUint64(1 * d.GlobalProps().GetPerTicketWeight()))
	globalPostReward := new(big.Int).SetUint64(d.GlobalProps().GetPoolPostRewards().Value)
	bigTotalPostWeight := StringToBigInt(d.GlobalProps().GetWeightedVpsPost())
	decayedPostWeight := bigDecay(bigTotalPostWeight)
	exceptNextBlockPostWeightedVps := decayedPostWeight.Add(decayedPostWeight, postWeight)
	bigGlobalPostReward := globalPostReward.Add(globalPostReward, new(big.Int).SetUint64(perBlockPostReward(d)))
	reward := ProportionAlgorithm(postWeight, exceptNextBlockPostWeightedVps, bigGlobalPostReward)
	exceptGlobalClaimRewardAfterCashout := d.GlobalProps().GetClaimedPostRewards().Add(&prototype.Vest{Value: reward.Uint64()})
	//exceptGlobalRewardAfterCashout := new(big.Int).Sub(bigGlobalPostReward, reward)
	exceptGlobalRewardAfterCashout := &prototype.Vest{ Value: new(big.Int).Sub(bigGlobalPostReward, reward).Uint64()}

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.Post(1).GetTicket(), uint32(1))
	a.Equal(d.GlobalProps().GetWeightedVpsPost(), exceptNextBlockPostWeightedVps.String() )
	vest1 := d.Account(tester.acc0.Name).GetVest().Value
	realReward := vest1 - vest0
	a.Equal(reward.Uint64(), realReward)
	a.Equal(d.Post(POST).GetRewards().Value, realReward)
	a.Equal(d.Post(POST).GetCashoutBlockNum(), app.CashoutCompleted)
	a.Equal(d.GlobalProps().GetClaimedPostRewards(), exceptGlobalClaimRewardAfterCashout)
	a.Equal(d.GlobalProps().GetPoolPostRewards(), exceptGlobalRewardAfterCashout)
}