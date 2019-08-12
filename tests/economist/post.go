package economist

import (
	"github.com/coschain/contentos-go/app/annual_mint"
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

	a := assert.New(t)
	a.NoError(tester.acc2.SendTrxAndProduceBlock(TransferToVest(tester.acc2.Name, tester.acc2.Name, constants.MinBpRegisterVest)))
	a.NoError(tester.acc2.SendTrxAndProduceBlock(BpRegister(tester.acc2.Name, "", "", tester.acc2.GetPubKey(), mintProps)))

	t.Run("normal", d.Test(tester.normal))
}

func (tester *PostTester) Test2(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	a := assert.New(t)
	a.NoError(tester.acc2.SendTrxAndProduceBlock(TransferToVest(tester.acc2.Name, tester.acc2.Name, constants.MinBpRegisterVest)))
	a.NoError(tester.acc2.SendTrxAndProduceBlock(BpRegister(tester.acc2.Name, "", "", tester.acc2.GetPubKey(), mintProps)))

	t.Run("cashout", d.Test(tester.cashout))
	t.Run("cashout after other cashout", d.Test(tester.cashoutAfterOtherCashout))
	t.Run("mul cashout", d.Test(tester.multiCashout))
}

func (tester *PostTester) Test3(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	a := assert.New(t)
	a.NoError(tester.acc2.SendTrxAndProduceBlock(TransferToVest(tester.acc2.Name, tester.acc2.Name, constants.MinBpRegisterVest)))
	a.NoError(tester.acc2.SendTrxAndProduceBlock(BpRegister(tester.acc2.Name, "", "", tester.acc2.GetPubKey(), mintProps)))

	t.Run("huge global vp", d.Test(tester.hugeGlobalVp))
	t.Run("zero global vp", d.Test(tester.zeroGlobalVp))
}

func (tester *PostTester) Test4(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	a := assert.New(t)
	a.NoError(tester.acc2.SendTrxAndProduceBlock(TransferToVest(tester.acc2.Name, tester.acc2.Name, constants.MinBpRegisterVest)))
	a.NoError(tester.acc2.SendTrxAndProduceBlock(BpRegister(tester.acc2.Name, "", "", tester.acc2.GetPubKey(), mintProps)))

	t.Run("with ticket", d.Test(tester.withTicket))
}

//func ISqrt(n string) *big.Int {
//	bigInt := new(big.Int)
//	value, _ := bigInt.SetString(n, 10)
//	sqrt := bigInt.Sqrt(value)
//	return sqrt
//}

func StringToBigInt(n string) *big.Int {
	bigInt := new(big.Int)
	value, _ := bigInt.SetString(n, 10)
	return value
}

func ProportionAlgorithm(numerator *big.Int, denominator *big.Int, total *big.Int) *big.Int {
	if denominator.Cmp(new(big.Int).SetUint64(0)) == 0 {
		return new(big.Int).SetUint64(0)
	} else {
		numeratorMul := new(big.Int).Mul(numerator, total)
		result := new(big.Int).Div(numeratorMul, denominator)
		return result
	}
}

func perBlockPostReward(d *Dandelion) uint64 {
	ith := d.GlobalProps().GetIthYear()
	annualBudget := annual_mint.CalculateBudget(ith)
	blockCurrency := annual_mint.CalculatePerBlockBudget(annualBudget)

	creatorReward := blockCurrency * constants.RewardRateCreator / constants.PERCENT
	postReward := creatorReward * constants.RewardRateAuthor / constants.PERCENT
	return postReward
}

func perBlockPostDappReward(d *Dandelion) uint64 {
	ith := d.GlobalProps().GetIthYear()
	annualBudget := annual_mint.CalculateBudget(ith)
	blockCurrency := annual_mint.CalculatePerBlockBudget(annualBudget)

	dappReward := blockCurrency * constants.RewardRateDapp / constants.PERCENT
	replyDappReward := dappReward * constants.RewardRateReply / constants.PERCENT
	postDappReward := dappReward - replyDappReward
	return postDappReward
}

func decay(rawValue uint64) uint64 {
	value := rawValue - rawValue * constants.BlockInterval / constants.VpDecayTime
	return value
}

func bigDecay(rawValue *big.Int) *big.Int {
	var decayValue big.Int
	decayValue.Mul(rawValue, new(big.Int).SetUint64(constants.BlockInterval))
	decayValue.Div(&decayValue, new(big.Int).SetUint64(constants.VpDecayTime))
	rawValue.Sub(rawValue, &decayValue)
	return rawValue
}

func (tester *PostTester) normal(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100
	const VEST = 1000

	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(1, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	post1Block := d.GlobalProps().GetHeadBlockNumber() - 1
	post1Cashout := post1Block + constants.PostCashOutDelayBlock
	a.Equal(post1Cashout, d.Post(1).GetCashoutBlockNum())
	a.NoError(tester.acc0.SendTrx(TransferToVest(tester.acc0.Name, tester.acc0.Name, VEST)))
	a.NoError(tester.acc1.SendTrx(TransferToVest(tester.acc1.Name, tester.acc1.Name, VEST)))
	// waiting for vp charge
	// next block post will be cashout
	a.NoError(d.ProduceBlocks(BLOCKS))
	vest0 := d.Account(tester.acc0.Name).GetVest().Value
	a.NoError(tester.acc0.SendTrx(Vote(tester.acc0.Name, 1)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - BLOCKS - 1))
	a.Equal(d.Account(tester.acc0.Name).GetVest().Value, vest0)

	// to cashout
	a.NoError(d.ProduceBlocks(1))
	a.NotEqual(d.Account(tester.acc0.Name).GetVest().Value, vest0)

	vest1 := d.Account(tester.acc0.Name).GetVest().Value
	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.Account(tester.acc0.Name).GetVest().Value, vest1)
}

func (tester *PostTester) cashout(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100
	const VEST = 1000

	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(1, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	post1Block := d.GlobalProps().GetHeadBlockNumber() - 1
	post1Cashout := post1Block + constants.PostCashOutDelayBlock
	a.Equal(post1Cashout, d.Post(1).GetCashoutBlockNum())
	a.NoError(tester.acc0.SendTrx(TransferToVest(tester.acc0.Name, tester.acc0.Name, VEST)))
	a.NoError(tester.acc1.SendTrx(TransferToVest(tester.acc1.Name, tester.acc1.Name, VEST)))

	a.NoError(d.ProduceBlocks(BLOCKS))
	vest0 := d.Account(tester.acc0.Name).GetVest().Value
	a.NoError(tester.acc0.SendTrx(Vote(tester.acc0.Name, 1)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - BLOCKS - 1))

	postWeight := StringToBigInt(d.Post(1).GetWeightedVp())
	globalPostReward := new(big.Int).SetUint64(d.GlobalProps().GetPostRewards().Value)
	globalPostDappReward := new(big.Int).SetUint64(d.GlobalProps().GetPostDappRewards().Value)
	bigTotalPostWeight, _ := new(big.Int).SetString(d.GlobalProps().GetPostWeightedVps(), 10)
	decayedPostWeight := bigDecay(bigTotalPostWeight)
	exceptNextBlockPostWeightedVps := decayedPostWeight.Add(decayedPostWeight, postWeight)
	bigGlobalPostReward := globalPostReward.Add(globalPostReward, new(big.Int).SetUint64(perBlockPostReward(d)))
	bigGlobalPostDappReward := globalPostDappReward.Add(globalPostDappReward, new(big.Int).SetUint64(perBlockPostDappReward(d)))
	pr1 := new(big.Int).Mul(postWeight, bigGlobalPostReward)
	exceptPostReward := new(big.Int).Div(pr1, exceptNextBlockPostWeightedVps)
	pr2 := new(big.Int).Mul(postWeight, bigGlobalPostDappReward)
	exceptPostDappReward := new(big.Int).Div(pr2, exceptNextBlockPostWeightedVps)

	reward := new(big.Int).Add(exceptPostReward, exceptPostDappReward).Uint64()

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetPostWeightedVps(), exceptNextBlockPostWeightedVps.String())
	vest1 := d.Account(tester.acc0.Name).GetVest().Value
	realReward := vest1 - vest0
	a.Equal(reward, realReward)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *PostTester) cashoutAfterOtherCashout(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100

	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(2, tester.acc0.Name, "title", "content", []string{"2"}, nil)))
	post1Block := d.GlobalProps().GetHeadBlockNumber() - 1
	post1Cashout := post1Block + constants.PostCashOutDelayBlock
	a.Equal(post1Cashout, d.Post(2).GetCashoutBlockNum())

	a.NoError(d.ProduceBlocks(BLOCKS))
	vest0 := d.Account(tester.acc0.Name).GetVest().Value
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, 2)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - BLOCKS - 1))

	postWeight := StringToBigInt(d.Post(2).GetWeightedVp())
	globalPostReward := new(big.Int).SetUint64(d.GlobalProps().GetPostRewards().Value)
	globalPostDappReward := new(big.Int).SetUint64(d.GlobalProps().GetPostDappRewards().Value)
	bigTotalPostWeight, _ := new(big.Int).SetString(d.GlobalProps().GetPostWeightedVps(), 10)
	decayedPostWeight := bigDecay(bigTotalPostWeight)
	exceptNextBlockPostWeightedVps := decayedPostWeight.Add(decayedPostWeight, postWeight)
	bigGlobalPostReward := globalPostReward.Add(globalPostReward, new(big.Int).SetUint64(perBlockPostReward(d)))
	bigGlobalPostDappReward := globalPostDappReward.Add(globalPostDappReward, new(big.Int).SetUint64(perBlockPostDappReward(d)))
	pr1 := new(big.Int).Mul(postWeight, bigGlobalPostReward)
	exceptPostReward := new(big.Int).Div(pr1, exceptNextBlockPostWeightedVps)
	pr2 := new(big.Int).Mul(postWeight, bigGlobalPostDappReward)
	exceptPostDappReward := new(big.Int).Div(pr2, exceptNextBlockPostWeightedVps)
	reward := new(big.Int).Add(exceptPostReward, exceptPostDappReward).Uint64()

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetPostWeightedVps(), exceptNextBlockPostWeightedVps.String())
	vest1 := d.Account(tester.acc0.Name).GetVest().Value
	realReward := vest1 - vest0
	a.Equal(reward, realReward)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *PostTester) multiCashout(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100

	a.NoError(tester.acc0.SendTrx(Post(3, tester.acc0.Name, "title", "content", []string{"3"}, nil)))
	a.NoError(tester.acc1.SendTrx(Post(4, tester.acc1.Name, "title", "content", []string{"4"}, nil)))
	a.NoError(d.ProduceBlocks(1))

	a.NoError(d.ProduceBlocks(BLOCKS))
	vestold3 := d.Account(tester.acc0.Name).GetVest().Value
	vestold4 := d.Account(tester.acc1.Name).GetVest().Value
	a.NoError(tester.acc0.SendTrx(Vote(tester.acc0.Name, 4)))
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, 3)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - BLOCKS - 1))

	// convert to uint64 to make test easier
	// the mul result less than uint64.MAX
	post3Weight :=  StringToBigInt(d.Post(3).GetWeightedVp())
	post4Weight :=  StringToBigInt(d.Post(4).GetWeightedVp())
	postsWeight := new(big.Int).Add(post3Weight, post4Weight)

	globalPostReward := new(big.Int).SetUint64(d.GlobalProps().GetPostRewards().Value)
	globalPostDappReward := new(big.Int).SetUint64(d.GlobalProps().GetPostDappRewards().Value)
	bigTotalPostWeight, _ := new(big.Int).SetString(d.GlobalProps().GetPostWeightedVps(), 10)

	decayedPostWeight := bigDecay(bigTotalPostWeight)
	exceptNextBlockPostWeightedVps := decayedPostWeight.Add(decayedPostWeight, postsWeight)
	bigGlobalPostReward := globalPostReward.Add(globalPostReward, new(big.Int).SetUint64(perBlockPostReward(d)))
	bigGlobalPostDappReward := globalPostDappReward.Add(globalPostDappReward, new(big.Int).SetUint64(perBlockPostDappReward(d)))
	pr1 := new(big.Int).Mul(postsWeight, bigGlobalPostReward)
	exceptPostReward := new(big.Int).Div(pr1, exceptNextBlockPostWeightedVps)
	post3Reward := ProportionAlgorithm(post3Weight, postsWeight, exceptPostReward)
	post4Reward := ProportionAlgorithm(post4Weight, postsWeight, exceptPostReward)
	pr2 := new(big.Int).Mul(postsWeight, bigGlobalPostDappReward)
	exceptPostDappReward := new(big.Int).Div(pr2, exceptNextBlockPostWeightedVps)
	post3DappReward := ProportionAlgorithm(post3Weight, postsWeight, exceptPostDappReward)
	post4DappReward := ProportionAlgorithm(post4Weight, postsWeight, exceptPostDappReward)
	reward3 := post3Reward.Uint64() + post3DappReward.Uint64()
	reward4 := post4Reward.Uint64() + post4DappReward.Uint64()

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetPostWeightedVps(), exceptNextBlockPostWeightedVps.String())
	vestnew3 := d.Account(tester.acc0.Name).GetVest().Value
	vestnew4 := d.Account(tester.acc1.Name).GetVest().Value
	real3Reward := vestnew3 - vestold3
	real4Reward := vestnew4 - vestold4
	a.Equal(reward3, real3Reward)
	a.Equal(reward4, real4Reward)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *PostTester) hugeGlobalVp(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100
	const VEST = 1000

	_ = d.ModifyProps(func(props *prototype.DynamicProperties) {
		maxUint64 := new(big.Int).SetUint64(math.MaxUint64)
		factor := new(big.Int).SetUint64(10)
		postWeightedVp := new(big.Int).Mul(maxUint64, factor)
		props.PostWeightedVps = postWeightedVp.String()
	})
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(1, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	a.NoError(tester.acc0.SendTrx(TransferToVest(tester.acc0.Name, tester.acc0.Name, VEST)))

	a.NoError(d.ProduceBlocks(BLOCKS))
	vest0 := d.Account(tester.acc0.Name).GetVest().Value
	a.NoError(tester.acc0.SendTrx(Vote(tester.acc0.Name, 1)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - BLOCKS - 1))

	postWeight := StringToBigInt(d.Post(1).GetWeightedVp())
	globalPostReward := new(big.Int).SetUint64(d.GlobalProps().GetPostRewards().Value)
	globalPostDappReward := new(big.Int).SetUint64(d.GlobalProps().GetPostDappRewards().Value)
	bigTotalPostWeight, _ := new(big.Int).SetString(d.GlobalProps().GetPostWeightedVps(), 10)
	decayedPostWeight := bigDecay(bigTotalPostWeight)
	exceptNextBlockPostWeightedVps := decayedPostWeight.Add(decayedPostWeight, postWeight)
	bigGlobalPostReward := globalPostReward.Add(globalPostReward, new(big.Int).SetUint64(perBlockPostReward(d)))
	bigGlobalPostDappReward := globalPostDappReward.Add(globalPostDappReward, new(big.Int).SetUint64(perBlockPostDappReward(d)))
	pr1 := new(big.Int).Mul(postWeight, bigGlobalPostReward)
	exceptPostReward := new(big.Int).Div(pr1, exceptNextBlockPostWeightedVps)
	pr2 := new(big.Int).Mul(postWeight, bigGlobalPostDappReward)
	exceptPostDappReward := new(big.Int).Div(pr2, exceptNextBlockPostWeightedVps)

	reward := new(big.Int).Add(exceptPostReward, exceptPostDappReward)

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetPostWeightedVps(), exceptNextBlockPostWeightedVps.String() )
	vest1 := d.Account(tester.acc0.Name).GetVest().Value
	realReward := vest1 - vest0
	a.Equal(reward.Uint64(), realReward)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *PostTester) zeroGlobalVp(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100

	_ = d.ModifyProps(func(props *prototype.DynamicProperties) {
		postWeightedVp := new(big.Int).SetUint64(0)
		props.PostWeightedVps = postWeightedVp.String()
	})
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(2, tester.acc0.Name, "title", "content", []string{"2"}, nil)))

	a.NoError(d.ProduceBlocks(BLOCKS))
	vest0 := d.Account(tester.acc0.Name).GetVest().Value
	a.NoError(tester.acc0.SendTrx(Vote(tester.acc0.Name, 2)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - BLOCKS - 1))

	postWeight := StringToBigInt(d.Post(2).GetWeightedVp())
	globalPostReward := new(big.Int).SetUint64(d.GlobalProps().GetPostRewards().Value)
	globalPostDappReward := new(big.Int).SetUint64(d.GlobalProps().GetPostDappRewards().Value)
	bigTotalPostWeight, _ := new(big.Int).SetString(d.GlobalProps().GetPostWeightedVps(), 10)
	decayedPostWeight := bigDecay(bigTotalPostWeight)
	exceptNextBlockPostWeightedVps := decayedPostWeight.Add(decayedPostWeight, postWeight)
	bigGlobalPostReward := globalPostReward.Add(globalPostReward, new(big.Int).SetUint64(perBlockPostReward(d)))
	bigGlobalPostDappReward := globalPostDappReward.Add(globalPostDappReward, new(big.Int).SetUint64(perBlockPostDappReward(d)))
	pr1 := new(big.Int).Mul(postWeight, bigGlobalPostReward)
	exceptPostReward := new(big.Int).Div(pr1, exceptNextBlockPostWeightedVps)
	pr2 := new(big.Int).Mul(postWeight, bigGlobalPostDappReward)
	exceptPostDappReward := new(big.Int).Div(pr2, exceptNextBlockPostWeightedVps)

	reward := new(big.Int).Add(exceptPostReward, exceptPostDappReward)

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetPostWeightedVps(), exceptNextBlockPostWeightedVps.String() )
	vest1 := d.Account(tester.acc0.Name).GetVest().Value
	realReward := vest1 - vest0
	a.Equal(reward.Uint64(), realReward)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *PostTester) withTicket(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100
	const VEST = 10 * constants.COSTokenDecimals

	a.NoError(tester.acc0.SendTrxAndProduceBlock(TransferToVest(tester.acc0.Name, tester.acc0.Name, VEST)))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(AcquireTicket(tester.acc0.Name, 1)))

	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(1, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	a.NoError(tester.acc0.SendTrx(VoteByTicket(tester.acc0.Name, 1, 1)))

	a.NoError(d.ProduceBlocks(BLOCKS))
	vest0 := d.Account(tester.acc0.Name).GetVest().Value
	a.NoError(tester.acc0.SendTrx(Vote(tester.acc0.Name, 1)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - BLOCKS - 1))

	postWeight := StringToBigInt(d.Post(1).GetWeightedVp())
	postWeight = new(big.Int).Add(postWeight, new(big.Int).SetUint64(1 * d.GlobalProps().GetPerTicketWeight()))
	globalPostReward := new(big.Int).SetUint64(d.GlobalProps().GetPostRewards().Value)
	globalPostDappReward := new(big.Int).SetUint64(d.GlobalProps().GetPostDappRewards().Value)
	bigTotalPostWeight, _ := new(big.Int).SetString(d.GlobalProps().GetPostWeightedVps(), 10)
	decayedPostWeight := bigDecay(bigTotalPostWeight)
	exceptNextBlockPostWeightedVps := decayedPostWeight.Add(decayedPostWeight, postWeight)
	bigGlobalPostReward := globalPostReward.Add(globalPostReward, new(big.Int).SetUint64(perBlockPostReward(d)))
	bigGlobalPostDappReward := globalPostDappReward.Add(globalPostDappReward, new(big.Int).SetUint64(perBlockPostDappReward(d)))
	pr1 := new(big.Int).Mul(postWeight, bigGlobalPostReward)
	exceptPostReward := new(big.Int).Div(pr1, exceptNextBlockPostWeightedVps)
	pr2 := new(big.Int).Mul(postWeight, bigGlobalPostDappReward)
	exceptPostDappReward := new(big.Int).Div(pr2, exceptNextBlockPostWeightedVps)

	reward := new(big.Int).Add(exceptPostReward, exceptPostDappReward)

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.Post(1).GetTicket(), uint32(1))
	a.Equal(d.GlobalProps().GetPostWeightedVps(), exceptNextBlockPostWeightedVps.String() )
	vest1 := d.Account(tester.acc0.Name).GetVest().Value
	realReward := vest1 - vest0
	a.Equal(reward.Uint64(), realReward)
}