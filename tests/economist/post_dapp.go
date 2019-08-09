package economist

import (
	"github.com/coschain/contentos-go/common/constants"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

type PostDappTester struct {
	acc0,acc1,acc2,acc3,acc4 *DandelionAccount
}

func (tester *PostDappTester) Test(t *testing.T, d *Dandelion) {
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

func (tester *PostDappTester) normal1(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100
	const POST = 1

	beneficiary := []map[string]int{{tester.acc0.Name: 10000}}
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, beneficiary)))
	post1Block := d.GlobalProps().GetHeadBlockNumber() - 1
	post1Cashout := post1Block + constants.PostCashOutDelayBlock
	a.Equal(post1Cashout, d.Post(POST).GetCashoutBlockNum())

	a.NoError(d.ProduceBlocks(BLOCKS))
	acc0vest0 := d.Account(tester.acc0.Name).GetVest().Value

	a.NoError(tester.acc0.SendTrx(Vote(tester.acc0.Name, POST)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - BLOCKS - 1))

	postWeight := StringToBigInt(d.Post(POST).GetWeightedVp())
	globalPostReward := new(big.Int).SetUint64(d.GlobalProps().GetPostRewards().Value)
	globalPostDappReward := new(big.Int).SetUint64(d.GlobalProps().GetPostDappRewards().Value)
	bigTotalPostWeight, _ := new(big.Int).SetString(d.GlobalProps().GetPostWeightedVps(), 10)
	decayedPostWeight := bigDecay(bigTotalPostWeight)
	exceptNextBlockPostWeightedVps := decayedPostWeight.Add(decayedPostWeight, postWeight)
	bigGlobalPostReward := globalPostReward.Add(globalPostReward, new(big.Int).SetUint64(perBlockPostReward(d)))
	bigGlobalPostDappReward := globalPostDappReward.Add(globalPostDappReward, new(big.Int).SetUint64(perBlockPostDappReward(d)))
	pr1 := new(big.Int).Mul(postWeight, bigGlobalPostReward)
	exceptPostReward := new(big.Int).Div(pr1, exceptNextBlockPostWeightedVps).Uint64()
	pr2 := new(big.Int).Mul(postWeight, bigGlobalPostDappReward)
	exceptPostDappReward := new(big.Int).Div(pr2, exceptNextBlockPostWeightedVps).Uint64()
	reward := exceptPostReward + exceptPostDappReward

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetPostWeightedVps(), exceptNextBlockPostWeightedVps.String())
	acc0vest1 := d.Account(tester.acc0.Name).GetVest().Value
	a.NotZero(reward)
	a.Equal(reward, acc0vest1 - acc0vest0)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *PostDappTester) normal2(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100
	const POST = 2

	beneficiary := []map[string]int{{tester.acc0.Name: 5000}}
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"2"}, beneficiary)))
	post1Block := d.GlobalProps().GetHeadBlockNumber() - 1
	post1Cashout := post1Block + constants.PostCashOutDelayBlock
	a.Equal(post1Cashout, d.Post(POST).GetCashoutBlockNum())

	a.NoError(d.ProduceBlocks(BLOCKS))
	acc0vest0 := d.Account(tester.acc0.Name).GetVest().Value

	a.NoError(tester.acc0.SendTrx(Vote(tester.acc0.Name, POST)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - BLOCKS - 1))

	postWeight := StringToBigInt(d.Post(POST).GetWeightedVp())
	globalPostReward := new(big.Int).SetUint64(d.GlobalProps().GetPostRewards().Value)
	globalPostDappReward := new(big.Int).SetUint64(d.GlobalProps().GetPostDappRewards().Value)
	bigTotalPostWeight, _ := new(big.Int).SetString(d.GlobalProps().GetPostWeightedVps(), 10)
	decayedPostWeight := bigDecay(bigTotalPostWeight)
	exceptNextBlockPostWeightedVps := decayedPostWeight.Add(decayedPostWeight, postWeight)
	bigGlobalPostReward := globalPostReward.Add(globalPostReward, new(big.Int).SetUint64(perBlockPostReward(d)))
	bigGlobalPostDappReward := globalPostDappReward.Add(globalPostDappReward, new(big.Int).SetUint64(perBlockPostDappReward(d)))
	pr1 := new(big.Int).Mul(postWeight, bigGlobalPostReward)
	exceptPostReward := new(big.Int).Div(pr1, exceptNextBlockPostWeightedVps).Uint64()
	pr2 := new(big.Int).Mul(postWeight, bigGlobalPostDappReward)
	exceptPostDappReward := new(big.Int).Div(pr2, exceptNextBlockPostWeightedVps).Uint64()

	reward := exceptPostReward + exceptPostDappReward

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetPostWeightedVps(), exceptNextBlockPostWeightedVps.String())
	acc0vest1 := d.Account(tester.acc0.Name).GetVest().Value
	a.NotZero(reward)
	a.Equal(reward, acc0vest1 - acc0vest0)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *PostDappTester) normal3(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100
	const POST = 3

	beneficiary := []map[string]int{{tester.acc1.Name: 10000}}
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"3"}, beneficiary)))
	post1Block := d.GlobalProps().GetHeadBlockNumber() - 1
	post1Cashout := post1Block + constants.PostCashOutDelayBlock
	a.Equal(post1Cashout, d.Post(POST).GetCashoutBlockNum())

	a.NoError(d.ProduceBlocks(BLOCKS))
	acc0vest0 := d.Account(tester.acc0.Name).GetVest().Value
	acc1vest0 := d.Account(tester.acc1.Name).GetVest().Value

	a.NoError(tester.acc0.SendTrx(Vote(tester.acc0.Name, POST)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - BLOCKS - 1))

	postWeight := StringToBigInt(d.Post(POST).GetWeightedVp())
	globalPostReward := new(big.Int).SetUint64(d.GlobalProps().GetPostRewards().Value)
	globalPostDappReward := new(big.Int).SetUint64(d.GlobalProps().GetPostDappRewards().Value)
	bigTotalPostWeight, _ := new(big.Int).SetString(d.GlobalProps().GetPostWeightedVps(), 10)
	decayedPostWeight := bigDecay(bigTotalPostWeight)
	exceptNextBlockPostWeightedVps := decayedPostWeight.Add(decayedPostWeight, postWeight)
	bigGlobalPostReward := globalPostReward.Add(globalPostReward, new(big.Int).SetUint64(perBlockPostReward(d)))
	bigGlobalPostDappReward := globalPostDappReward.Add(globalPostDappReward, new(big.Int).SetUint64(perBlockPostDappReward(d)))
	pr1 := new(big.Int).Mul(postWeight, bigGlobalPostReward)
	exceptPostReward := new(big.Int).Div(pr1, exceptNextBlockPostWeightedVps).Uint64()
	pr2 := new(big.Int).Mul(postWeight, bigGlobalPostDappReward)
	exceptPostDappReward := new(big.Int).Div(pr2, exceptNextBlockPostWeightedVps).Uint64()

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetPostWeightedVps(), exceptNextBlockPostWeightedVps.String())
	acc0vest1 := d.Account(tester.acc0.Name).GetVest().Value
	acc1vest1 := d.Account(tester.acc1.Name).GetVest().Value
	a.NotZero(exceptPostReward)
	a.NotZero(exceptPostDappReward)
	a.Equal(exceptPostReward, acc0vest1 - acc0vest0)
	a.Equal(exceptPostDappReward, acc1vest1 - acc1vest0)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *PostDappTester) normal4(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100
	const POST = 4

	beneficiary := []map[string]int{{tester.acc0.Name: 5000}, {tester.acc1.Name: 5000}}
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"4"}, beneficiary)))
	post1Block := d.GlobalProps().GetHeadBlockNumber() - 1
	post1Cashout := post1Block + constants.PostCashOutDelayBlock
	a.Equal(post1Cashout, d.Post(POST).GetCashoutBlockNum())

	a.NoError(d.ProduceBlocks(BLOCKS))
	acc0vest0 := d.Account(tester.acc0.Name).GetVest().Value
	acc1vest0 := d.Account(tester.acc1.Name).GetVest().Value

	a.NoError(tester.acc0.SendTrx(Vote(tester.acc0.Name, POST)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - BLOCKS - 1))

	postWeight := StringToBigInt(d.Post(POST).GetWeightedVp())
	globalPostReward := new(big.Int).SetUint64(d.GlobalProps().GetPostRewards().Value)
	globalPostDappReward := new(big.Int).SetUint64(d.GlobalProps().GetPostDappRewards().Value)
	bigTotalPostWeight, _ := new(big.Int).SetString(d.GlobalProps().GetPostWeightedVps(), 10)
	decayedPostWeight := bigDecay(bigTotalPostWeight)
	exceptNextBlockPostWeightedVps := decayedPostWeight.Add(decayedPostWeight, postWeight)
	bigGlobalPostReward := globalPostReward.Add(globalPostReward, new(big.Int).SetUint64(perBlockPostReward(d)))
	bigGlobalPostDappReward := globalPostDappReward.Add(globalPostDappReward, new(big.Int).SetUint64(perBlockPostDappReward(d)))
	pr1 := new(big.Int).Mul(postWeight, bigGlobalPostReward)
	exceptPostReward := new(big.Int).Div(pr1, exceptNextBlockPostWeightedVps).Uint64()
	pr2 := new(big.Int).Mul(postWeight, bigGlobalPostDappReward)
	exceptPostDappReward := new(big.Int).Div(pr2, exceptNextBlockPostWeightedVps).Uint64()

	reward1 := exceptPostReward + exceptPostDappReward - exceptPostDappReward / 2
	reward2 := exceptPostDappReward / 2

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetPostWeightedVps(), exceptNextBlockPostWeightedVps.String())
	acc0vest1 := d.Account(tester.acc0.Name).GetVest().Value
	acc1vest1 := d.Account(tester.acc1.Name).GetVest().Value
	a.NotZero(reward1)
	a.NotZero(reward2)
	a.Equal(reward1, acc0vest1 - acc0vest0)
	a.Equal(reward2, acc1vest1 - acc1vest0)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *PostDappTester) normal5(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100
	const POST = 5

	beneficiary := []map[string]int{{tester.acc0.Name: 5000}, {tester.acc1.Name: 2000}, {tester.acc2.Name: 3000}}
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"5"}, beneficiary)))
	post1Block := d.GlobalProps().GetHeadBlockNumber() - 1
	post1Cashout := post1Block + constants.PostCashOutDelayBlock
	a.Equal(post1Cashout, d.Post(POST).GetCashoutBlockNum())

	a.NoError(d.ProduceBlocks(BLOCKS))
	acc0vest0 := d.Account(tester.acc0.Name).GetVest().Value
	acc1vest0 := d.Account(tester.acc1.Name).GetVest().Value
	acc2vest0 := d.Account(tester.acc2.Name).GetVest().Value

	a.NoError(tester.acc0.SendTrx(Vote(tester.acc0.Name, POST)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - BLOCKS - 1))

	postWeight := StringToBigInt(d.Post(POST).GetWeightedVp())
	globalPostReward := new(big.Int).SetUint64(d.GlobalProps().GetPostRewards().Value)
	globalPostDappReward := new(big.Int).SetUint64(d.GlobalProps().GetPostDappRewards().Value)
	bigTotalPostWeight, _ := new(big.Int).SetString(d.GlobalProps().GetPostWeightedVps(), 10)
	decayedPostWeight := bigDecay(bigTotalPostWeight)
	exceptNextBlockPostWeightedVps := decayedPostWeight.Add(decayedPostWeight, postWeight)
	bigGlobalPostReward := globalPostReward.Add(globalPostReward, new(big.Int).SetUint64(perBlockPostReward(d)))
	bigGlobalPostDappReward := globalPostDappReward.Add(globalPostDappReward, new(big.Int).SetUint64(perBlockPostDappReward(d)))
	pr1 := new(big.Int).Mul(postWeight, bigGlobalPostReward)
	exceptPostReward := new(big.Int).Div(pr1, exceptNextBlockPostWeightedVps).Uint64()
	pr2 := new(big.Int).Mul(postWeight, bigGlobalPostDappReward)
	exceptPostDappReward := new(big.Int).Div(pr2, exceptNextBlockPostWeightedVps).Uint64()

	//reward := exceptPostReward + exceptPostDappReward
	reward2 := exceptPostDappReward * 2 / 10
	reward3 := exceptPostDappReward * 3 / 10
	reward1 := exceptPostReward + exceptPostDappReward - reward2 - reward3

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetPostWeightedVps(), exceptNextBlockPostWeightedVps.String())
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

func (tester *PostDappTester) normal6(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100
	const POST = 6

	beneficiary := []map[string]int{{tester.acc0.Name: 5000}, {tester.acc1.Name: 4000}}
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"6"}, beneficiary)))
	post1Block := d.GlobalProps().GetHeadBlockNumber() - 1
	post1Cashout := post1Block + constants.PostCashOutDelayBlock
	a.Equal(post1Cashout, d.Post(POST).GetCashoutBlockNum())

	a.NoError(d.ProduceBlocks(BLOCKS))
	acc0vest0 := d.Account(tester.acc0.Name).GetVest().Value
	acc1vest0 := d.Account(tester.acc1.Name).GetVest().Value

	a.NoError(tester.acc0.SendTrx(Vote(tester.acc0.Name, POST)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - BLOCKS - 1))

	postWeight := StringToBigInt(d.Post(POST).GetWeightedVp())
	globalPostReward := new(big.Int).SetUint64(d.GlobalProps().GetPostRewards().Value)
	globalPostDappReward := new(big.Int).SetUint64(d.GlobalProps().GetPostDappRewards().Value)
	bigTotalPostWeight, _ := new(big.Int).SetString(d.GlobalProps().GetPostWeightedVps(), 10)
	decayedPostWeight := bigDecay(bigTotalPostWeight)
	exceptNextBlockPostWeightedVps := decayedPostWeight.Add(decayedPostWeight, postWeight)
	bigGlobalPostReward := globalPostReward.Add(globalPostReward, new(big.Int).SetUint64(perBlockPostReward(d)))
	bigGlobalPostDappReward := globalPostDappReward.Add(globalPostDappReward, new(big.Int).SetUint64(perBlockPostDappReward(d)))
	pr1 := new(big.Int).Mul(postWeight, bigGlobalPostReward)
	exceptPostReward := new(big.Int).Div(pr1, exceptNextBlockPostWeightedVps).Uint64()
	pr2 := new(big.Int).Mul(postWeight, bigGlobalPostDappReward)
	exceptPostDappReward := new(big.Int).Div(pr2, exceptNextBlockPostWeightedVps).Uint64()
	//reward := exceptPostReward + exceptPostDappReward
	reward2 := exceptPostDappReward * 4 / 10
	reward1 := exceptPostReward + exceptPostDappReward - reward2
	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetPostWeightedVps(), exceptNextBlockPostWeightedVps.String())
	acc0vest1 := d.Account(tester.acc0.Name).GetVest().Value
	acc1vest1 := d.Account(tester.acc1.Name).GetVest().Value
	a.NotZero(reward1)
	a.NotZero(reward2)

	a.Equal(reward1, acc0vest1 - acc0vest0)
	a.Equal(reward2, acc1vest1 - acc1vest0)
	// make all post/test has been cashouted
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
}

func (tester *PostDappTester) normal7(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100
	const POST = 7

	beneficiary := []map[string]int{{tester.acc0.Name: 5000}, {tester.acc1.Name: 4000}, {tester.acc2.Name: 2000}}
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"7"}, beneficiary)))
	post1Block := d.GlobalProps().GetHeadBlockNumber() - 1
	post1Cashout := post1Block + constants.PostCashOutDelayBlock
	a.Equal(post1Cashout, d.Post(POST).GetCashoutBlockNum())

	a.NoError(d.ProduceBlocks(BLOCKS))


	a.NoError(tester.acc0.SendTrx(Vote(tester.acc0.Name, POST)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - BLOCKS - 1))

	acc0vest0 := d.Account(tester.acc0.Name).GetVest().Value
	acc1vest0 := d.Account(tester.acc1.Name).GetVest().Value
	acc2vest0 := d.Account(tester.acc2.Name).GetVest().Value

	postWeight := StringToBigInt(d.Post(POST).GetWeightedVp())
	globalPostReward := new(big.Int).SetUint64(d.GlobalProps().GetPostRewards().Value)
	globalPostDappReward := new(big.Int).SetUint64(d.GlobalProps().GetPostDappRewards().Value)
	bigTotalPostWeight, _ := new(big.Int).SetString(d.GlobalProps().GetPostWeightedVps(), 10)
	decayedPostWeight := bigDecay(bigTotalPostWeight)
	exceptNextBlockPostWeightedVps := decayedPostWeight.Add(decayedPostWeight, postWeight)
	bigGlobalPostReward := globalPostReward.Add(globalPostReward, new(big.Int).SetUint64(perBlockPostReward(d)))
	bigGlobalPostDappReward := globalPostDappReward.Add(globalPostDappReward, new(big.Int).SetUint64(perBlockPostDappReward(d)))
	pr1 := new(big.Int).Mul(postWeight, bigGlobalPostReward)
	exceptPostReward := new(big.Int).Div(pr1, exceptNextBlockPostWeightedVps).Uint64()
	pr2 := new(big.Int).Mul(postWeight, bigGlobalPostDappReward)
	exceptPostDappReward := new(big.Int).Div(pr2, exceptNextBlockPostWeightedVps).Uint64()
	//reward := exceptPostReward + exceptPostDappReward
	reward2 := exceptPostDappReward * 4 / 10
	reward3 := exceptPostDappReward * 0
	reward1 := exceptPostReward + exceptPostDappReward - reward2

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetPostWeightedVps(), exceptNextBlockPostWeightedVps.String() )
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