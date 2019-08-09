package economist

import (
	"github.com/coschain/contentos-go/common/constants"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

type ReplyDappTester struct {
	acc0,acc1,acc2,acc3,acc4 *DandelionAccount
}

func (tester *ReplyDappTester) Test(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")
	tester.acc3 = d.Account("actor3")
	tester.acc4 = d.Account("actor4")

	a := assert.New(t)
	a.NoError(tester.acc4.SendTrxAndProduceBlock(TransferToVest(tester.acc4.Name, tester.acc4.Name, constants.MinBpRegisterVest)))
	a.NoError(tester.acc4.SendTrxAndProduceBlock(BpRegister(tester.acc4.Name, "", "", tester.acc4.GetPubKey(), mintProps)))

	const VEST = 1000

	a.NoError(tester.acc0.SendTrx(TransferToVest(tester.acc0.Name, tester.acc0.Name, VEST)))

	t.Run("normal self 100%", d.Test(tester.normal1))
	t.Run("normal self 50%", d.Test(tester.normal2))
	t.Run("normal other 100%", d.Test(tester.normal3))
	t.Run("normal self and other half-and-half", d.Test(tester.normal4))
	t.Run("normal three people", d.Test(tester.normal5))
	t.Run("normal two people less 100%", d.Test(tester.normal6))
	t.Run("three people greater than 100%", d.Test(tester.normal7))
}

func (tester *ReplyDappTester) normal1(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100
	const POST = 1
	const REPLY = 2

	beneficiary := []map[string]int{{tester.acc0.Name: 10000}}
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Reply(REPLY, POST,  tester.acc0.Name, "content",  beneficiary)))
	post1Block := d.GlobalProps().GetHeadBlockNumber() - 1
	post1Cashout := post1Block + constants.PostCashOutDelayBlock
	a.Equal(post1Cashout, d.Post(REPLY).GetCashoutBlockNum())

	a.NoError(d.ProduceBlocks(BLOCKS))
	acc0vest0 := d.Account(tester.acc0.Name).GetVest().Value

	a.NoError(tester.acc0.SendTrx(Vote(tester.acc0.Name, REPLY)))
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
	exceptReplyReward := new(big.Int).Div(pr1, exceptNextBlockReplyWeightedVps).Uint64()
	pr2 := new(big.Int).Mul(replyWeight, bigGlobalReplyDappReward)
	exceptReplyDappReward := new(big.Int).Div(pr2, exceptNextBlockReplyWeightedVps).Uint64()
	reward := exceptReplyReward + exceptReplyDappReward

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetReplyWeightedVps(), exceptNextBlockReplyWeightedVps.String())
	acc0vest1 := d.Account(tester.acc0.Name).GetVest().Value
	realReward := acc0vest1 - acc0vest0
	a.Equal(reward, realReward)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *ReplyDappTester) normal2(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100
	const POST = 3
	const REPLY = 4

	beneficiary := []map[string]int{{tester.acc0.Name: 5000}}
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"2"}, nil)))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Reply(REPLY, POST,  tester.acc0.Name, "content",  beneficiary)))

	a.NoError(d.ProduceBlocks(BLOCKS))
	acc0vest0 := d.Account(tester.acc0.Name).GetVest().Value

	a.NoError(tester.acc0.SendTrx(Vote(tester.acc0.Name, REPLY)))
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
	exceptReplyReward := new(big.Int).Div(pr1, exceptNextBlockReplyWeightedVps).Uint64()
	pr2 := new(big.Int).Mul(replyWeight, bigGlobalReplyDappReward)
	exceptReplyDappReward := new(big.Int).Div(pr2, exceptNextBlockReplyWeightedVps).Uint64()
	reward := exceptReplyReward + exceptReplyDappReward

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetReplyWeightedVps(), exceptNextBlockReplyWeightedVps.String())
	acc0vest1 := d.Account(tester.acc0.Name).GetVest().Value
	realReward := acc0vest1 - acc0vest0
	a.Equal(reward, realReward)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *ReplyDappTester) normal3(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100
	const POST = 5
	const REPLY = 6

	beneficiary := []map[string]int{{tester.acc1.Name: 10000}}
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"3"}, nil)))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Reply(REPLY, POST,  tester.acc0.Name, "content",  beneficiary)))

	a.NoError(d.ProduceBlocks(BLOCKS))
	acc0vest0 := d.Account(tester.acc0.Name).GetVest().Value
	acc1vest0 := d.Account(tester.acc1.Name).GetVest().Value

	a.NoError(tester.acc0.SendTrx(Vote(tester.acc0.Name, REPLY)))
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
	exceptReplyReward := new(big.Int).Div(pr1, exceptNextBlockReplyWeightedVps).Uint64()
	pr2 := new(big.Int).Mul(replyWeight, bigGlobalReplyDappReward)
	exceptReplyDappReward := new(big.Int).Div(pr2, exceptNextBlockReplyWeightedVps).Uint64()

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetReplyWeightedVps(), exceptNextBlockReplyWeightedVps.String())
	acc0vest1 := d.Account(tester.acc0.Name).GetVest().Value
	acc1vest1 := d.Account(tester.acc1.Name).GetVest().Value
	a.NotZero(exceptReplyReward)
	a.NotZero(exceptReplyDappReward)
	a.Equal(exceptReplyReward, acc0vest1 - acc0vest0)
	a.Equal(exceptReplyDappReward, acc1vest1 - acc1vest0)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *ReplyDappTester) normal4(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100
	const POST = 7
	const REPLY = 8

	beneficiary := []map[string]int{{tester.acc0.Name: 5000}, {tester.acc1.Name: 5000}}
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"4"}, nil)))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Reply(REPLY, POST,  tester.acc0.Name, "content",  beneficiary)))

	a.NoError(d.ProduceBlocks(BLOCKS))
	acc0vest0 := d.Account(tester.acc0.Name).GetVest().Value
	acc1vest0 := d.Account(tester.acc1.Name).GetVest().Value

	a.NoError(tester.acc0.SendTrx(Vote(tester.acc0.Name, REPLY)))
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
	exceptReplyReward := new(big.Int).Div(pr1, exceptNextBlockReplyWeightedVps).Uint64()
	pr2 := new(big.Int).Mul(replyWeight, bigGlobalReplyDappReward)
	exceptReplyDappReward := new(big.Int).Div(pr2, exceptNextBlockReplyWeightedVps).Uint64()

	//reward := exceptPostReward + exceptPostDappReward
	reward1 := exceptReplyReward + exceptReplyDappReward - exceptReplyDappReward / 2
	reward2 := exceptReplyDappReward / 2

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetReplyWeightedVps(), exceptNextBlockReplyWeightedVps.String())
	acc0vest1 := d.Account(tester.acc0.Name).GetVest().Value
	acc1vest1 := d.Account(tester.acc1.Name).GetVest().Value
	a.NotZero(reward1)
	a.NotZero(reward2)
	a.Equal(reward1, acc0vest1 - acc0vest0)
	a.Equal(reward2, acc1vest1 - acc1vest0)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *ReplyDappTester) normal5(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100
	const POST = 9
	const REPLY = 10

	beneficiary := []map[string]int{{tester.acc0.Name: 5000}, {tester.acc1.Name: 2000}, {tester.acc2.Name: 3000}}
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"5"}, nil)))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Reply(REPLY, POST,  tester.acc0.Name, "content",  beneficiary)))

	a.NoError(d.ProduceBlocks(BLOCKS))
	acc0vest0 := d.Account(tester.acc0.Name).GetVest().Value
	acc1vest0 := d.Account(tester.acc1.Name).GetVest().Value
	acc2vest0 := d.Account(tester.acc2.Name).GetVest().Value

	a.NoError(tester.acc0.SendTrx(Vote(tester.acc0.Name, REPLY)))
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
	exceptReplyReward := new(big.Int).Div(pr1, exceptNextBlockReplyWeightedVps).Uint64()
	pr2 := new(big.Int).Mul(replyWeight, bigGlobalReplyDappReward)
	exceptReplyDappReward := new(big.Int).Div(pr2, exceptNextBlockReplyWeightedVps).Uint64()

	reward2 := exceptReplyDappReward * 2 / 10
	reward3 := exceptReplyDappReward * 3 / 10
	reward1 := exceptReplyReward + exceptReplyDappReward - reward2 - reward3

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetReplyWeightedVps(), exceptNextBlockReplyWeightedVps.String())
	acc0vest1 := d.Account(tester.acc0.Name).GetVest().Value
	acc1vest1 := d.Account(tester.acc1.Name).GetVest().Value
	acc2vest1 := d.Account(tester.acc2.Name).GetVest().Value
	a.NotZero(reward1)
	a.NotZero(reward2)
	a.NotZero(reward3)
	a.Equal(reward1, acc0vest1 - acc0vest0)
	a.Equal(reward2, acc1vest1 - acc1vest0)
	a.Equal(reward3, acc2vest1 - acc2vest0)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *ReplyDappTester) normal6(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100
	const POST = 11
	const REPLY = 12

	beneficiary := []map[string]int{{tester.acc0.Name: 5000}, {tester.acc1.Name: 4000}}
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"6"}, nil)))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Reply(REPLY, POST,  tester.acc0.Name, "content",  beneficiary)))

	a.NoError(d.ProduceBlocks(BLOCKS))
	acc0vest0 := d.Account(tester.acc0.Name).GetVest().Value
	acc1vest0 := d.Account(tester.acc1.Name).GetVest().Value

	a.NoError(tester.acc0.SendTrx(Vote(tester.acc0.Name, REPLY)))
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
	exceptReplyReward := new(big.Int).Div(pr1, exceptNextBlockReplyWeightedVps).Uint64()
	pr2 := new(big.Int).Mul(replyWeight, bigGlobalReplyDappReward)
	exceptReplyDappReward := new(big.Int).Div(pr2, exceptNextBlockReplyWeightedVps).Uint64()

	reward2 := exceptReplyDappReward * 4 / 10
	reward1 := exceptReplyReward + exceptReplyDappReward - reward2

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetReplyWeightedVps(), exceptNextBlockReplyWeightedVps.String())
	acc0vest1 := d.Account(tester.acc0.Name).GetVest().Value
	acc1vest1 := d.Account(tester.acc1.Name).GetVest().Value
	a.NotZero(reward1)
	a.NotZero(reward2)
	a.Equal(reward1, acc0vest1 - acc0vest0)
	a.Equal(reward2, acc1vest1 - acc1vest0)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *ReplyDappTester) normal7(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100
	const POST = 13
	const REPLY = 14

	beneficiary := []map[string]int{{tester.acc0.Name: 5000}, {tester.acc1.Name: 4000}, {tester.acc2.Name: 2000}}
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"7"}, nil)))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Reply(REPLY, POST,  tester.acc0.Name, "content",  beneficiary)))

	a.NoError(d.ProduceBlocks(BLOCKS))
	acc0vest0 := d.Account(tester.acc0.Name).GetVest().Value
	acc1vest0 := d.Account(tester.acc1.Name).GetVest().Value
	acc2vest0 := d.Account(tester.acc2.Name).GetVest().Value

	a.NoError(tester.acc0.SendTrx(Vote(tester.acc0.Name, REPLY)))
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
	exceptReplyReward := new(big.Int).Div(pr1, exceptNextBlockReplyWeightedVps).Uint64()
	pr2 := new(big.Int).Mul(replyWeight, bigGlobalReplyDappReward)
	exceptReplyDappReward := new(big.Int).Div(pr2, exceptNextBlockReplyWeightedVps).Uint64()

	reward2 := exceptReplyDappReward * 4 / 10
	reward3 := exceptReplyDappReward * 0
	reward1 := exceptReplyReward + exceptReplyDappReward - reward2

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetReplyWeightedVps(), exceptNextBlockReplyWeightedVps.String())
	acc0vest1 := d.Account(tester.acc0.Name).GetVest().Value
	acc1vest1 := d.Account(tester.acc1.Name).GetVest().Value
	acc2vest1 := d.Account(tester.acc2.Name).GetVest().Value
	a.NotZero(reward1)
	a.NotZero(reward2)
	a.Equal(reward1, acc0vest1 - acc0vest0)
	a.Equal(reward2, acc1vest1 - acc1vest0)
	a.Equal(reward3, acc2vest1 - acc2vest0)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}