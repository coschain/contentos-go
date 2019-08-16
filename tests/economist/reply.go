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

type ReplyTester struct {
	acc0,acc1,acc2 *DandelionAccount
}

func (tester *ReplyTester) Test1(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	a := assert.New(t)
	a.NoError(tester.acc2.SendTrxAndProduceBlock(TransferToVest(tester.acc2.Name, tester.acc2.Name, constants.MinBpRegisterVest, "")))
	a.NoError(tester.acc2.SendTrxAndProduceBlock(BpRegister(tester.acc2.Name, "", "", tester.acc2.GetPubKey(), mintProps)))

	t.Run("normal", d.Test(tester.normal))
}

func (tester *ReplyTester) Test2(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	a := assert.New(t)
	a.NoError(tester.acc2.SendTrxAndProduceBlock(TransferToVest(tester.acc2.Name, tester.acc2.Name, constants.MinBpRegisterVest, "")))
	a.NoError(tester.acc2.SendTrxAndProduceBlock(BpRegister(tester.acc2.Name, "", "", tester.acc2.GetPubKey(), mintProps)))

	t.Run("cashout", d.Test(tester.cashout))
	t.Run("cashout after other cashout", d.Test(tester.cashoutAfterOtherCashout))
	t.Run("mul cashout", d.Test(tester.multiCashout))
}

func (tester *ReplyTester) Test3(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	a := assert.New(t)
	a.NoError(tester.acc2.SendTrxAndProduceBlock(TransferToVest(tester.acc2.Name, tester.acc2.Name, constants.MinBpRegisterVest, "")))
	a.NoError(tester.acc2.SendTrxAndProduceBlock(BpRegister(tester.acc2.Name, "", "", tester.acc2.GetPubKey(), mintProps)))

	t.Run("huge global vp", d.Test(tester.hugeGlobalVp))
	t.Run("zero global vp", d.Test(tester.zeroGlobalVp))
}

func (tester *ReplyTester) Test4(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	a := assert.New(t)
	a.NoError(tester.acc2.SendTrxAndProduceBlock(TransferToVest(tester.acc2.Name, tester.acc2.Name, constants.MinBpRegisterVest, "")))
	a.NoError(tester.acc2.SendTrxAndProduceBlock(BpRegister(tester.acc2.Name, "", "", tester.acc2.GetPubKey(), mintProps)))

	t.Run("with ticket", d.Test(tester.withTicket))
}

func perBlockReplyReward(d *Dandelion) uint64 {
	ith := d.GlobalProps().GetIthYear()
	annualBudget := annual_mint.CalculateBudget(ith)
	blockCurrency := annual_mint.CalculatePerBlockBudget(annualBudget)

	creatorReward := blockCurrency * constants.RewardRateCreator / constants.PERCENT
	postReward := creatorReward * constants.RewardRateReply / constants.PERCENT
	return postReward
}

func perBlockReplyDappReward(d *Dandelion) uint64 {
	ith := d.GlobalProps().GetIthYear()
	annualBudget := annual_mint.CalculateBudget(ith)
	blockCurrency := annual_mint.CalculatePerBlockBudget(annualBudget)

	dappReward := blockCurrency * constants.RewardRateDapp / constants.PERCENT
	replyDappReward := dappReward * constants.RewardRateReply / constants.PERCENT
	return replyDappReward
}

func (tester *ReplyTester) normal(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100
	const VEST = 1000
	const POST = 1
	const REPLY = 2

	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	post1Block := d.GlobalProps().GetHeadBlockNumber() - 1
	post1Cashout := post1Block + constants.PostCashOutDelayBlock
	a.Equal(post1Cashout, d.Post(POST).GetCashoutBlockNum())
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Reply(REPLY, POST,  tester.acc0.Name, "content",  nil)))
	a.NoError(tester.acc0.SendTrx(TransferToVest(tester.acc0.Name, tester.acc0.Name, VEST, "")))
	a.NoError(tester.acc1.SendTrx(TransferToVest(tester.acc1.Name, tester.acc1.Name, VEST, "")))
	// waiting for vp charge
	// next block post will be cashout
	a.NoError(d.ProduceBlocks(BLOCKS))
	vest0 := d.Account(tester.acc0.Name).GetVest().Value
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, REPLY)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - BLOCKS - 1))
	a.Equal(d.Account(tester.acc0.Name).GetVest().Value, vest0)

	// to cashout
	a.NoError(d.ProduceBlocks(1))
	a.NotEqual(d.Account(tester.acc0.Name).GetVest().Value, vest0)

	vest1 := d.Account(tester.acc0.Name).GetVest().Value
	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.Account(tester.acc0.Name).GetVest().Value, vest1)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *ReplyTester) cashout(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100
	const VEST = 1000
	const POST = 1
	const REPLY = 2

	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	post1Block := d.GlobalProps().GetHeadBlockNumber() - 1
	post1Cashout := post1Block + constants.PostCashOutDelayBlock
	a.Equal(post1Cashout, d.Post(1).GetCashoutBlockNum())
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Reply(REPLY, POST,  tester.acc0.Name, "content",  nil)))
	a.NoError(tester.acc0.SendTrx(TransferToVest(tester.acc0.Name, tester.acc0.Name, VEST, "")))
	a.NoError(tester.acc1.SendTrx(TransferToVest(tester.acc1.Name, tester.acc1.Name, VEST, "")))

	a.NoError(d.ProduceBlocks(BLOCKS))
	vest0 := d.Account(tester.acc0.Name).GetVest().Value
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, REPLY)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - BLOCKS - 1))

	replyWeight := StringToBigInt(d.Post(REPLY).GetWeightedVp())
	globalReplyReward := new(big.Int).SetUint64(d.GlobalProps().GetReplyRewards().Value)
	globalReplyDappReward := new(big.Int).SetUint64(d.GlobalProps().GetReplyDappRewards().Value)
	bigTotalReplyWeight, _ := new(big.Int).SetString(d.GlobalProps().GetReplyWeightedVps(), 10)
	decayedReplyWeight := bigDecay(bigTotalReplyWeight)
	exceptNextBlockReplyWeightedVps := decayedReplyWeight.Add(decayedReplyWeight, replyWeight)
	bigGlobalReplyReward := globalReplyReward.Add(globalReplyReward, new(big.Int).SetUint64(perBlockReplyReward(d)))
	bigGlobalReplyDappReward := globalReplyDappReward.Add(globalReplyDappReward, new(big.Int).SetUint64(perBlockReplyDappReward(d)))
	pr1 := new(big.Int).Mul(replyWeight, bigGlobalReplyReward)
	exceptReplyReward := new(big.Int).Div(pr1, exceptNextBlockReplyWeightedVps)
	pr2 := new(big.Int).Mul(replyWeight, bigGlobalReplyDappReward)
	exceptReplyDappReward := new(big.Int).Div(pr2, exceptNextBlockReplyWeightedVps)
	reward := exceptReplyReward.Uint64() + exceptReplyDappReward.Uint64()

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetReplyWeightedVps(), exceptNextBlockReplyWeightedVps.String())
	vest1 := d.Account(tester.acc0.Name).GetVest().Value
	realReward := vest1 - vest0
	a.Equal(reward, realReward)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *ReplyTester) cashoutAfterOtherCashout(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100
	const POST = 3
	const REPLY = 4

	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"2"}, nil)))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Reply(REPLY, POST,  tester.acc0.Name, "content",  nil)))

	a.NoError(d.ProduceBlocks(BLOCKS))
	vest0 := d.Account(tester.acc0.Name).GetVest().Value
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, REPLY)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - BLOCKS - 1))

	replyWeight := StringToBigInt(d.Post(REPLY).GetWeightedVp())
	globalReplyReward := new(big.Int).SetUint64(d.GlobalProps().GetReplyRewards().Value)
	globalReplyDappReward := new(big.Int).SetUint64(d.GlobalProps().GetReplyDappRewards().Value)
	bigTotalReplyWeight, _ := new(big.Int).SetString(d.GlobalProps().GetReplyWeightedVps(), 10)
	decayedReplyWeight := bigDecay(bigTotalReplyWeight)
	exceptNextBlockReplyWeightedVps := decayedReplyWeight.Add(decayedReplyWeight, replyWeight)
	bigGlobalReplyReward := globalReplyReward.Add(globalReplyReward, new(big.Int).SetUint64(perBlockReplyReward(d)))
	bigGlobalReplyDappReward := globalReplyDappReward.Add(globalReplyDappReward, new(big.Int).SetUint64(perBlockReplyDappReward(d)))
	pr1 := new(big.Int).Mul(replyWeight, bigGlobalReplyReward)
	exceptReplyReward := new(big.Int).Div(pr1, exceptNextBlockReplyWeightedVps)
	pr2 := new(big.Int).Mul(replyWeight, bigGlobalReplyDappReward)
	exceptReplyDappReward := new(big.Int).Div(pr2, exceptNextBlockReplyWeightedVps)
	reward := exceptReplyReward.Uint64() + exceptReplyDappReward.Uint64()

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetReplyWeightedVps(), exceptNextBlockReplyWeightedVps.String())
	vest1 := d.Account(tester.acc0.Name).GetVest().Value
	realReward := vest1 - vest0
	a.Equal(reward, realReward)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *ReplyTester) multiCashout(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100
	const POST = 5
	const REPLY1 = 6
	const REPLY2 = 7

	a.NoError(tester.acc0.SendTrx(Post(POST, tester.acc0.Name, "title", "content", []string{"3"}, nil)))
	a.NoError(d.ProduceBlocks(1))
	a.NoError(tester.acc0.SendTrx(Reply(REPLY1, POST,  tester.acc0.Name, "content1",  nil)))
	a.NoError(tester.acc1.SendTrx(Reply(REPLY2, POST,  tester.acc1.Name, "content2",  nil)))
	a.NoError(d.ProduceBlocks(1))

	a.NoError(d.ProduceBlocks(BLOCKS))
	vestold1 := d.Account(tester.acc0.Name).GetVest().Value
	vestold2 := d.Account(tester.acc1.Name).GetVest().Value
	a.NoError(tester.acc0.SendTrx(Vote(tester.acc0.Name, REPLY2)))
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, REPLY1)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - BLOCKS - 1))

	// convert to uint64 to make test easier
	// the mul result less than uint64.MAX
	reply1Weight :=  StringToBigInt(d.Post(REPLY1).GetWeightedVp())
	reply2Weight :=  StringToBigInt(d.Post(REPLY2).GetWeightedVp())

	globalReplyReward := new(big.Int).SetUint64(d.GlobalProps().GetReplyRewards().Value)
	globalReplyDappReward := new(big.Int).SetUint64(d.GlobalProps().GetReplyDappRewards().Value)
	bigTotalReplyWeight, _ := new(big.Int).SetString(d.GlobalProps().GetReplyWeightedVps(), 10)

	repliesWeight := new(big.Int).Add(reply1Weight, reply2Weight)
	decayedReplyWeight := bigDecay(bigTotalReplyWeight)
	exceptNextBlockReplyWeightedVps := decayedReplyWeight.Add(decayedReplyWeight, repliesWeight)
	bigGlobalReplyReward := globalReplyReward.Add(globalReplyReward, new(big.Int).SetUint64(perBlockReplyReward(d)))
	bigGlobalReplyDappReward := globalReplyDappReward.Add(globalReplyDappReward, new(big.Int).SetUint64(perBlockReplyDappReward(d)))
	pr1 := new(big.Int).Mul(repliesWeight, bigGlobalReplyReward)
	exceptReplyReward := new(big.Int).Div(pr1, exceptNextBlockReplyWeightedVps)
	reply1Reward := ProportionAlgorithm(reply1Weight, repliesWeight, exceptReplyReward)
	reply2Reward := ProportionAlgorithm(reply2Weight, repliesWeight, exceptReplyReward)
	pr2 := new(big.Int).Mul(repliesWeight, bigGlobalReplyDappReward)
	exceptReplyDappReward := new(big.Int).Div(pr2, exceptNextBlockReplyWeightedVps)
	reply1DappReward := ProportionAlgorithm(reply1Weight, repliesWeight, exceptReplyDappReward)
	reply2DappReward := ProportionAlgorithm(reply2Weight, repliesWeight, exceptReplyDappReward)
	reward1 := reply1Reward.Uint64() + reply1DappReward.Uint64()
	reward2 := reply2Reward.Uint64() + reply2DappReward.Uint64()

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetReplyWeightedVps(), exceptNextBlockReplyWeightedVps.String())
	vestnew1 := d.Account(tester.acc0.Name).GetVest().Value
	vestnew2 := d.Account(tester.acc1.Name).GetVest().Value
	real1Reward := vestnew1 - vestold1
	real2Reward := vestnew2 - vestold2
	a.Equal(reward1, real1Reward)
	a.Equal(reward2, real2Reward)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *ReplyTester) hugeGlobalVp(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100
	const VEST = 1000
	const POST = 1
	const REPLY = 2

	_ = d.ModifyProps(func(props *prototype.DynamicProperties) {
		maxUint64 := new(big.Int).SetUint64(math.MaxUint64)
		factor := new(big.Int).SetUint64(10)
		replyWeightedVp := new(big.Int).Mul(maxUint64, factor)
		props.ReplyWeightedVps = replyWeightedVp.String()
	})
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Reply(REPLY, POST,  tester.acc0.Name, "content",  nil)))
	a.NoError(tester.acc0.SendTrx(TransferToVest(tester.acc0.Name, tester.acc0.Name, VEST, "")))

	a.NoError(d.ProduceBlocks(BLOCKS))
	vest0 := d.Account(tester.acc0.Name).GetVest().Value
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, REPLY)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - BLOCKS - 1))

	replyWeight := StringToBigInt(d.Post(REPLY).GetWeightedVp())
	globalReplyReward := new(big.Int).SetUint64(d.GlobalProps().GetReplyRewards().Value)
	globalReplyDappReward := new(big.Int).SetUint64(d.GlobalProps().GetReplyDappRewards().Value)
	bigTotalReplyWeight, _ := new(big.Int).SetString(d.GlobalProps().GetReplyWeightedVps(), 10)
	decayedReplyWeight := bigDecay(bigTotalReplyWeight)
	exceptNextBlockReplyWeightedVps := decayedReplyWeight.Add(decayedReplyWeight, replyWeight)
	bigGlobalReplyReward := globalReplyReward.Add(globalReplyReward, new(big.Int).SetUint64(perBlockReplyReward(d)))
	bigGlobalReplyDappReward := globalReplyDappReward.Add(globalReplyDappReward, new(big.Int).SetUint64(perBlockReplyDappReward(d)))
	pr1 := new(big.Int).Mul(replyWeight, bigGlobalReplyReward)
	exceptReplyReward := new(big.Int).Div(pr1, exceptNextBlockReplyWeightedVps)
	pr2 := new(big.Int).Mul(replyWeight, bigGlobalReplyDappReward)
	exceptReplyDappReward := new(big.Int).Div(pr2, exceptNextBlockReplyWeightedVps)

	reward := new(big.Int).Add(exceptReplyReward, exceptReplyDappReward)

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetReplyWeightedVps(), exceptNextBlockReplyWeightedVps.String() )
	vest1 := d.Account(tester.acc0.Name).GetVest().Value
	realReward := vest1 - vest0
	a.Equal(reward.Uint64(), realReward)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *ReplyTester) zeroGlobalVp(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100
	const VEST = 1000
	const POST = 3
	const REPLY = 4

	_ = d.ModifyProps(func(props *prototype.DynamicProperties) {
		replyWeightedVp := new(big.Int).SetUint64(0)
		props.ReplyWeightedVps = replyWeightedVp.String()
	})
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Reply(REPLY, POST,  tester.acc0.Name, "content",  nil)))

	a.NoError(d.ProduceBlocks(BLOCKS))
	vest0 := d.Account(tester.acc0.Name).GetVest().Value
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, REPLY)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - BLOCKS - 1))

	replyWeight := StringToBigInt(d.Post(REPLY).GetWeightedVp())
	globalReplyReward := new(big.Int).SetUint64(d.GlobalProps().GetReplyRewards().Value)
	globalReplyDappReward := new(big.Int).SetUint64(d.GlobalProps().GetReplyDappRewards().Value)
	bigTotalReplyWeight, _ := new(big.Int).SetString(d.GlobalProps().GetReplyWeightedVps(), 10)
	decayedReplyWeight := bigDecay(bigTotalReplyWeight)
	exceptNextBlockReplyWeightedVps := decayedReplyWeight.Add(decayedReplyWeight, replyWeight)
	bigGlobalReplyReward := globalReplyReward.Add(globalReplyReward, new(big.Int).SetUint64(perBlockReplyReward(d)))
	bigGlobalReplyDappReward := globalReplyDappReward.Add(globalReplyDappReward, new(big.Int).SetUint64(perBlockReplyDappReward(d)))
	pr1 := new(big.Int).Mul(replyWeight, bigGlobalReplyReward)
	exceptReplyReward := new(big.Int).Div(pr1, exceptNextBlockReplyWeightedVps)
	pr2 := new(big.Int).Mul(replyWeight, bigGlobalReplyDappReward)
	exceptReplyDappReward := new(big.Int).Div(pr2, exceptNextBlockReplyWeightedVps)

	reward := new(big.Int).Add(exceptReplyReward, exceptReplyDappReward)

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetReplyWeightedVps(), exceptNextBlockReplyWeightedVps.String() )
	vest1 := d.Account(tester.acc0.Name).GetVest().Value
	realReward := vest1 - vest0
	a.Equal(reward.Uint64(), realReward)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *ReplyTester) withTicket(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100
	const VEST = 10 * constants.COSTokenDecimals
	const POST = 1
	const REPLY = 2

	a.NoError(tester.acc0.SendTrxAndProduceBlock(TransferToVest(tester.acc0.Name, tester.acc0.Name, VEST, "")))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(AcquireTicket(tester.acc0.Name, 1)))

	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Reply(REPLY, POST,  tester.acc0.Name, "content",  nil)))
	a.NoError(tester.acc0.SendTrx(VoteByTicket(tester.acc0.Name, REPLY, 1)))


	a.NoError(d.ProduceBlocks(BLOCKS))
	vest0 := d.Account(tester.acc0.Name).GetVest().Value
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, REPLY)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - BLOCKS - 1))

	replyWeight := StringToBigInt(d.Post(REPLY).GetWeightedVp())
	replyWeight = new(big.Int).Add(replyWeight, new(big.Int).SetUint64(1 * d.GlobalProps().GetPerTicketWeight()))
	globalReplyReward := new(big.Int).SetUint64(d.GlobalProps().GetReplyRewards().Value)
	globalReplyDappReward := new(big.Int).SetUint64(d.GlobalProps().GetReplyDappRewards().Value)
	bigTotalReplyWeight, _ := new(big.Int).SetString(d.GlobalProps().GetReplyWeightedVps(), 10)
	decayedReplyWeight := bigDecay(bigTotalReplyWeight)
	exceptNextBlockReplyWeightedVps := decayedReplyWeight.Add(decayedReplyWeight, replyWeight)
	bigGlobalReplyReward := globalReplyReward.Add(globalReplyReward, new(big.Int).SetUint64(perBlockReplyReward(d)))
	bigGlobalReplyDappReward := globalReplyDappReward.Add(globalReplyDappReward, new(big.Int).SetUint64(perBlockReplyDappReward(d)))
	pr1 := new(big.Int).Mul(replyWeight, bigGlobalReplyReward)
	exceptReplyReward := new(big.Int).Div(pr1, exceptNextBlockReplyWeightedVps)
	pr2 := new(big.Int).Mul(replyWeight, bigGlobalReplyDappReward)
	exceptReplyDappReward := new(big.Int).Div(pr2, exceptNextBlockReplyWeightedVps)

	reward := new(big.Int).Add(exceptReplyReward, exceptReplyDappReward)

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetReplyWeightedVps(), exceptNextBlockReplyWeightedVps.String() )
	vest1 := d.Account(tester.acc0.Name).GetVest().Value
	realReward := vest1 - vest0
	a.Equal(reward.Uint64(), realReward)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}