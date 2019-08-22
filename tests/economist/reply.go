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

type ReplyTester struct {
	acc0,acc1,acc2 *DandelionAccount
}

func (tester *ReplyTester) Test1(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	registerBlockProducer(tester.acc2, t)

	const VEST = 1000
	SelfTransferToVesting([]*DandelionAccount{tester.acc0, tester.acc1}, VEST, t)

	t.Run("normal", d.Test(tester.normal))
}

func (tester *ReplyTester) Test2(t *testing.T, d *Dandelion) {
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

func (tester *ReplyTester) Test3(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	registerBlockProducer(tester.acc2, t)
	const VEST = 1000
	SelfTransferToVesting([]*DandelionAccount{tester.acc0, tester.acc1}, VEST, t)

	t.Run("huge global vp", d.Test(tester.hugeGlobalVp))
	t.Run("zero global vp", d.Test(tester.zeroGlobalVp))
}

func (tester *ReplyTester) Test4(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	registerBlockProducer(tester.acc2, t)

	t.Run("with ticket", d.Test(tester.withTicket))
}


func (tester *ReplyTester) normal(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const POST = 1
	const REPLY = 2

	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	post1Block := d.GlobalProps().GetHeadBlockNumber() - 1
	post1Cashout := post1Block + constants.PostCashOutDelayBlock
	a.Equal(post1Cashout, d.Post(POST).GetCashoutBlockNum())
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Reply(REPLY, POST,  tester.acc0.Name, "content",  nil)))

	// waiting for vp charge
	// next block post will be cashout
	vest0 := d.Account(tester.acc0.Name).GetVest().Value
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, REPLY)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - 2))
	a.Equal(d.Account(tester.acc0.Name).GetVest().Value, vest0)

	// to cashout
	a.NoError(d.ProduceBlocks(1))
	a.NotEqual(d.Account(tester.acc0.Name).GetVest().Value, vest0)

	vest1 := d.Account(tester.acc0.Name).GetVest().Value
	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.Account(tester.acc0.Name).GetVest().Value, vest1)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
	a.Equal(d.Post(REPLY).GetCashoutBlockNum(), app.CashoutCompleted)
}

func (tester *ReplyTester) cashout(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const POST = 1
	const REPLY = 2

	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	post1Block := d.GlobalProps().GetHeadBlockNumber() - 1
	post1Cashout := post1Block + constants.PostCashOutDelayBlock
	a.Equal(post1Cashout, d.Post(1).GetCashoutBlockNum())
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Reply(REPLY, POST,  tester.acc0.Name, "content",  nil)))

	vest0 := d.Account(tester.acc0.Name).GetVest().Value
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, REPLY)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - 2))

	replyWeight := StringToBigInt(d.Post(REPLY).GetWeightedVp())
	a.NotEqual(replyWeight.Int64(), int64(0))

	globalReplyReward := new(big.Int).SetUint64(d.GlobalProps().GetPoolReplyRewards().Value)
	bigTotalReplyWeight := StringToBigInt(d.GlobalProps().GetWeightedVpsReply())
	decayedReplyWeight := bigDecay(bigTotalReplyWeight)
	exceptNextBlockReplyWeightedVps := decayedReplyWeight.Add(decayedReplyWeight, replyWeight)
	bigGlobalReplyReward := globalReplyReward.Add(globalReplyReward, new(big.Int).SetUint64(perBlockReplyReward(d)))
	reward := ProportionAlgorithm(replyWeight, exceptNextBlockReplyWeightedVps, globalReplyReward)
	exceptGlobalClaimRewardAfterCashout := &prototype.Vest{Value: reward.Uint64()}
	exceptGlobalRewardAfterCashout := &prototype.Vest{ Value: new(big.Int).Sub(bigGlobalReplyReward, reward).Uint64()}

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetWeightedVpsReply(), exceptNextBlockReplyWeightedVps.String())
	vest1 := d.Account(tester.acc0.Name).GetVest().Value
	realReward := vest1 - vest0
	a.Equal(reward.Uint64(), realReward)
	a.Equal(d.GlobalProps().GetClaimedReplyRewards(), exceptGlobalClaimRewardAfterCashout)
	a.Equal(d.GlobalProps().GetPoolReplyRewards(), exceptGlobalRewardAfterCashout)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *ReplyTester) cashoutAfterOtherCashout(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const POST = 3
	const REPLY = 4

	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"2"}, nil)))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Reply(REPLY, POST,  tester.acc0.Name, "content",  nil)))

	vest0 := d.Account(tester.acc0.Name).GetVest().Value
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, REPLY)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - 2))

	replyWeight := StringToBigInt(d.Post(REPLY).GetWeightedVp())
	a.NotEqual(replyWeight.Int64(), int64(0))

	globalReplyReward := new(big.Int).SetUint64(d.GlobalProps().GetPoolReplyRewards().Value)
	bigTotalReplyWeight := StringToBigInt(d.GlobalProps().GetWeightedVpsReply())
	decayedReplyWeight := bigDecay(bigTotalReplyWeight)
	exceptNextBlockReplyWeightedVps := decayedReplyWeight.Add(decayedReplyWeight, replyWeight)
	bigGlobalReplyReward := globalReplyReward.Add(globalReplyReward, new(big.Int).SetUint64(perBlockReplyReward(d)))
	reward := ProportionAlgorithm(replyWeight, exceptNextBlockReplyWeightedVps, globalReplyReward)
	exceptGlobalClaimRewardAfterCashout := d.GlobalProps().ClaimedReplyRewards.Add(&prototype.Vest{Value: reward.Uint64()})
	exceptGlobalRewardAfterCashout := &prototype.Vest{ Value: new(big.Int).Sub(bigGlobalReplyReward, reward).Uint64()}

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetWeightedVpsReply(), exceptNextBlockReplyWeightedVps.String())
	vest1 := d.Account(tester.acc0.Name).GetVest().Value
	realReward := vest1 - vest0
	a.Equal(reward.Uint64(), realReward)
	a.Equal(d.GlobalProps().GetClaimedReplyRewards(), exceptGlobalClaimRewardAfterCashout)
	a.Equal(d.GlobalProps().GetPoolReplyRewards(), exceptGlobalRewardAfterCashout)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *ReplyTester) multiCashout(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const POST = 5
	const REPLY1 = 6
	const REPLY2 = 7
	const BLOCK = 100

	a.NoError(tester.acc0.SendTrx(Post(POST, tester.acc0.Name, "title", "content", []string{"3"}, nil)))
	// do not to interrupt reply cashout
	a.NoError(d.ProduceBlocks(1))

	a.NoError(tester.acc0.SendTrx(Reply(REPLY1, POST,  tester.acc0.Name, "content1",  nil)))
	a.NoError(tester.acc1.SendTrx(Reply(REPLY2, POST,  tester.acc1.Name, "content2",  nil)))
	a.NoError(d.ProduceBlocks(1))
	a.NoError(tester.acc0.SendTrx(Vote(tester.acc0.Name, REPLY2)))
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, REPLY1)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - 2))
	vestold1 := d.Account(tester.acc0.Name).GetVest().Value
	vestold2 := d.Account(tester.acc1.Name).GetVest().Value

	// convert to uint64 to make test easier
	// the mul result less than uint64.MAX
	reply1Weight :=  StringToBigInt(d.Post(REPLY1).GetWeightedVp())
	reply2Weight :=  StringToBigInt(d.Post(REPLY2).GetWeightedVp())
	a.NotEqual(reply1Weight.Int64(), int64(0))
	a.NotEqual(reply2Weight.Int64(), int64(0))

	globalReplyReward := new(big.Int).SetUint64(d.GlobalProps().GetPoolReplyRewards().Value)
	bigTotalReplyWeight := StringToBigInt(d.GlobalProps().GetWeightedVpsReply())

	repliesWeight := new(big.Int).Add(reply1Weight, reply2Weight)

	decayedReplyWeight := bigDecay(bigTotalReplyWeight)
	exceptNextBlockReplyWeightedVps := decayedReplyWeight.Add(decayedReplyWeight, repliesWeight)
	bigGlobalReplyReward := globalReplyReward.Add(globalReplyReward, new(big.Int).SetUint64(perBlockReplyReward(d)))
	reply1Reward := ProportionAlgorithm(reply1Weight, exceptNextBlockReplyWeightedVps, bigGlobalReplyReward)
	reply2Reward := ProportionAlgorithm(reply2Weight, exceptNextBlockReplyWeightedVps, bigGlobalReplyReward)
	repliesReward := new(big.Int).Add(reply1Reward, reply2Reward)
	exceptGlobalClaimRewardAfterCashout := d.GlobalProps().ClaimedReplyRewards.Add(&prototype.Vest{Value: repliesReward.Uint64()})
	exceptGlobalRewardAfterCashout := &prototype.Vest{ Value: new(big.Int).Sub(bigGlobalReplyReward, repliesReward).Uint64()}

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetWeightedVpsReply(), exceptNextBlockReplyWeightedVps.String())
	vestnew1 := d.Account(tester.acc0.Name).GetVest().Value
	vestnew2 := d.Account(tester.acc1.Name).GetVest().Value
	real1Reward := vestnew1 - vestold1
	real2Reward := vestnew2 - vestold2
	a.Equal(reply1Reward.Uint64(), real1Reward)
	a.Equal(reply2Reward.Uint64(), real2Reward)
	a.Equal(d.GlobalProps().GetClaimedReplyRewards(), exceptGlobalClaimRewardAfterCashout)
	a.Equal(d.GlobalProps().GetPoolReplyRewards(), exceptGlobalRewardAfterCashout)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *ReplyTester) hugeGlobalVp(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const POST = 1
	const REPLY = 2

	_ = d.ModifyProps(func(props *prototype.DynamicProperties) {
		maxUint64 := new(big.Int).SetUint64(math.MaxUint64)
		factor := new(big.Int).SetUint64(10)
		replyWeightedVp := new(big.Int).Mul(maxUint64, factor)
		props.WeightedVpsReply = replyWeightedVp.String()
	})
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Reply(REPLY, POST,  tester.acc0.Name, "content",  nil)))

	vest0 := d.Account(tester.acc0.Name).GetVest().Value
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, REPLY)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - 2))

	replyWeight := StringToBigInt(d.Post(REPLY).GetWeightedVp())
	a.NotEqual(replyWeight.Int64(), int64(0))

	globalReplyReward := new(big.Int).SetUint64(d.GlobalProps().GetPoolReplyRewards().Value)
	bigTotalReplyWeight := StringToBigInt(d.GlobalProps().GetWeightedVpsReply())
	decayedReplyWeight := bigDecay(bigTotalReplyWeight)
	exceptNextBlockReplyWeightedVps := decayedReplyWeight.Add(decayedReplyWeight, replyWeight)
	bigGlobalReplyReward := globalReplyReward.Add(globalReplyReward, new(big.Int).SetUint64(perBlockReplyReward(d)))
	reward := ProportionAlgorithm(replyWeight, exceptNextBlockReplyWeightedVps, globalReplyReward)
	exceptGlobalClaimRewardAfterCashout := &prototype.Vest{Value: reward.Uint64()}
	exceptGlobalRewardAfterCashout := &prototype.Vest{ Value: new(big.Int).Sub(bigGlobalReplyReward, reward).Uint64()}

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetWeightedVpsReply(), exceptNextBlockReplyWeightedVps.String() )
	vest1 := d.Account(tester.acc0.Name).GetVest().Value
	realReward := vest1 - vest0
	a.Equal(reward.Uint64(), realReward)
	a.Equal(d.GlobalProps().GetClaimedReplyRewards(), exceptGlobalClaimRewardAfterCashout)
	a.Equal(d.GlobalProps().GetPoolReplyRewards(), exceptGlobalRewardAfterCashout)
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
		props.WeightedVpsReply = replyWeightedVp.String()
	})
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Reply(REPLY, POST,  tester.acc0.Name, "content",  nil)))

	vest0 := d.Account(tester.acc0.Name).GetVest().Value
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, REPLY)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - 2))

	replyWeight := StringToBigInt(d.Post(REPLY).GetWeightedVp())
	a.NotEqual(replyWeight.Int64(), int64(0))

	globalReplyReward := new(big.Int).SetUint64(d.GlobalProps().GetPoolReplyRewards().Value)
	bigTotalReplyWeight := StringToBigInt(d.GlobalProps().GetWeightedVpsReply())
	decayedReplyWeight := bigDecay(bigTotalReplyWeight)
	exceptNextBlockReplyWeightedVps := decayedReplyWeight.Add(decayedReplyWeight, replyWeight)
	bigGlobalReplyReward := globalReplyReward.Add(globalReplyReward, new(big.Int).SetUint64(perBlockReplyReward(d)))
	reward := ProportionAlgorithm(replyWeight, exceptNextBlockReplyWeightedVps, globalReplyReward)
	exceptGlobalClaimRewardAfterCashout := &prototype.Vest{Value: reward.Uint64()}
	exceptGlobalRewardAfterCashout := &prototype.Vest{ Value: new(big.Int).Sub(bigGlobalReplyReward, reward).Uint64()}

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetWeightedVpsReply(), exceptNextBlockReplyWeightedVps.String() )
	vest1 := d.Account(tester.acc0.Name).GetVest().Value
	realReward := vest1 - vest0
	a.Equal(reward.Uint64(), realReward)
	a.Equal(d.GlobalProps().GetClaimedReplyRewards(), exceptGlobalClaimRewardAfterCashout)
	a.Equal(d.GlobalProps().GetPoolReplyRewards(), exceptGlobalRewardAfterCashout)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *ReplyTester) withTicket(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const VEST = 10 * constants.COSTokenDecimals
	const POST = 1
	const REPLY = 2

	a.NoError(tester.acc0.SendTrxAndProduceBlock(TransferToVest(tester.acc0.Name, tester.acc0.Name, VEST, "")))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(AcquireTicket(tester.acc0.Name, 1)))

	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Reply(REPLY, POST,  tester.acc0.Name, "content",  nil)))
	a.NoError(tester.acc0.SendTrx(VoteByTicket(tester.acc0.Name, REPLY, 1)))


	vest0 := d.Account(tester.acc0.Name).GetVest().Value
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, REPLY)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - 2))

	replyWeight := StringToBigInt(d.Post(REPLY).GetWeightedVp())
	a.NotEqual(replyWeight.Int64(), int64(0))

	replyWeight = new(big.Int).Add(replyWeight, new(big.Int).SetUint64(1 * d.GlobalProps().GetPerTicketWeight()))
	globalReplyReward := new(big.Int).SetUint64(d.GlobalProps().GetPoolReplyRewards().Value)
	bigTotalReplyWeight := StringToBigInt(d.GlobalProps().GetWeightedVpsReply())
	decayedReplyWeight := bigDecay(bigTotalReplyWeight)
	exceptNextBlockReplyWeightedVps := decayedReplyWeight.Add(decayedReplyWeight, replyWeight)
	bigGlobalReplyReward := globalReplyReward.Add(globalReplyReward, new(big.Int).SetUint64(perBlockReplyReward(d)))
	reward := ProportionAlgorithm(replyWeight, exceptNextBlockReplyWeightedVps, globalReplyReward)
	exceptGlobalClaimRewardAfterCashout := &prototype.Vest{Value: reward.Uint64()}
	exceptGlobalRewardAfterCashout := &prototype.Vest{ Value: new(big.Int).Sub(bigGlobalReplyReward, reward).Uint64()}

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetWeightedVpsReply(), exceptNextBlockReplyWeightedVps.String() )
	vest1 := d.Account(tester.acc0.Name).GetVest().Value
	realReward := vest1 - vest0
	a.Equal(reward.Uint64(), realReward)
	a.Equal(d.GlobalProps().GetClaimedReplyRewards(), exceptGlobalClaimRewardAfterCashout)
	a.Equal(d.GlobalProps().GetPoolReplyRewards(), exceptGlobalRewardAfterCashout)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}