package economist

import (
	"github.com/coschain/contentos-go/app/annual_mint"
	"github.com/coschain/contentos-go/common/constants"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/stretchr/testify/assert"
	"math/big"
	"strconv"
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

func ISqrt(n string) *big.Int {
	bigInt := new(big.Int)
	value, _ := bigInt.SetString(n, 10)
	sqrt := bigInt.Sqrt(value)
	return sqrt
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

func (tester *PostTester) normal(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100
	const VEST = 1000

	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(1, tester.acc0.Name, "title", "content", []string{"1"}, make(map[string]int))))
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

	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(1, tester.acc0.Name, "title", "content", []string{"1"}, make(map[string]int))))
	post1Block := d.GlobalProps().GetHeadBlockNumber() - 1
	post1Cashout := post1Block + constants.PostCashOutDelayBlock
	a.Equal(post1Cashout, d.Post(1).GetCashoutBlockNum())
	a.NoError(tester.acc0.SendTrx(TransferToVest(tester.acc0.Name, tester.acc0.Name, VEST)))
	a.NoError(tester.acc1.SendTrx(TransferToVest(tester.acc1.Name, tester.acc1.Name, VEST)))

	a.NoError(d.ProduceBlocks(BLOCKS))
	vest0 := d.Account(tester.acc0.Name).GetVest().Value
	a.NoError(tester.acc0.SendTrx(Vote(tester.acc0.Name, 1)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - BLOCKS - 1))

	// convert to uint64 to make test easier
	// the mul result less than uint64.MAX
	postWeight := ISqrt(d.Post(1).GetWeightedVp()).Uint64()
	globalPostReward := new(big.Int).SetUint64(d.GlobalProps().GetPostRewards().Value).Uint64()
	globalPostDappReward := new(big.Int).SetUint64(d.GlobalProps().GetPostDappRewards().Value).Uint64()
	bigTotalPostWeight, _ := new(big.Int).SetString(d.GlobalProps().GetPostWeightedVps(), 10)
	totalPostWeight := bigTotalPostWeight.Uint64()
	exceptNextBlockPostWeightedVps := decay(totalPostWeight) + postWeight
	exceptPostReward := postWeight * (globalPostReward + perBlockPostReward(d)) / exceptNextBlockPostWeightedVps
	exceptPostDappReward := postWeight * (globalPostDappReward + perBlockPostDappReward(d)) / exceptNextBlockPostWeightedVps
	reward := exceptPostReward + exceptPostDappReward

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetPostWeightedVps(), strconv.FormatUint(exceptNextBlockPostWeightedVps, 10))
	vest1 := d.Account(tester.acc0.Name).GetVest().Value
	realReward := vest1 - vest0
	a.Equal(reward, realReward)
}

func (tester *PostTester) cashoutAfterOtherCashout(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100

	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(2, tester.acc0.Name, "title", "content", []string{"2"}, make(map[string]int))))
	post1Block := d.GlobalProps().GetHeadBlockNumber() - 1
	post1Cashout := post1Block + constants.PostCashOutDelayBlock
	a.Equal(post1Cashout, d.Post(2).GetCashoutBlockNum())

	a.NoError(d.ProduceBlocks(BLOCKS))
	vest0 := d.Account(tester.acc0.Name).GetVest().Value
	a.NoError(tester.acc0.SendTrx(Vote(tester.acc0.Name, 2)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - BLOCKS - 1))

	// convert to uint64 to make test easier
	// the mul result less than uint64.MAX
	postWeight := ISqrt(d.Post(2).GetWeightedVp()).Uint64()
	globalPostReward := new(big.Int).SetUint64(d.GlobalProps().GetPostRewards().Value).Uint64()
	globalPostDappReward := new(big.Int).SetUint64(d.GlobalProps().GetPostDappRewards().Value).Uint64()
	bigTotalPostWeight, _ := new(big.Int).SetString(d.GlobalProps().GetPostWeightedVps(), 10)
	totalPostWeight := bigTotalPostWeight.Uint64()
	exceptNextBlockPostWeightedVps := decay(totalPostWeight) + postWeight
	exceptPostReward := postWeight * (globalPostReward + perBlockPostReward(d)) / exceptNextBlockPostWeightedVps
	exceptPostDappReward := postWeight * (globalPostDappReward + perBlockPostDappReward(d)) / exceptNextBlockPostWeightedVps
	reward := exceptPostReward + exceptPostDappReward

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetPostWeightedVps(), strconv.FormatUint(exceptNextBlockPostWeightedVps, 10))
	vest1 := d.Account(tester.acc0.Name).GetVest().Value
	realReward := vest1 - vest0
	a.Equal(reward, realReward)
}

func (tester *PostTester) multiCashout(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCKS = 100

	a.NoError(tester.acc0.SendTrx(Post(3, tester.acc0.Name, "title", "content", []string{"3"}, make(map[string]int))))
	a.NoError(tester.acc1.SendTrx(Post(4, tester.acc1.Name, "title", "content", []string{"4"}, make(map[string]int))))
	a.NoError(d.ProduceBlocks(1))

	a.NoError(d.ProduceBlocks(BLOCKS))
	vestold3 := d.Account(tester.acc0.Name).GetVest().Value
	vestold4 := d.Account(tester.acc1.Name).GetVest().Value
	a.NoError(tester.acc0.SendTrx(Vote(tester.acc0.Name, 4)))
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, 3)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - BLOCKS - 1))

	// convert to uint64 to make test easier
	// the mul result less than uint64.MAX
	post3Weight :=  ISqrt(d.Post(3).GetWeightedVp()).Uint64()
	post4Weight :=  ISqrt(d.Post(4).GetWeightedVp()).Uint64()

	globalPostReward := new(big.Int).SetUint64(d.GlobalProps().GetPostRewards().Value).Uint64()
	globalPostDappReward := new(big.Int).SetUint64(d.GlobalProps().GetPostDappRewards().Value).Uint64()
	bigTotalPostWeight, _ := new(big.Int).SetString(d.GlobalProps().GetPostWeightedVps(), 10)
	totalPostWeight := bigTotalPostWeight.Uint64()

	exceptNextBlockPostWeightedVps := decay(totalPostWeight) + post3Weight + post4Weight
	sumPostReward := (post3Weight + post4Weight) * (globalPostReward + perBlockPostReward(d)) / exceptNextBlockPostWeightedVps
	post3Reward := post3Weight * sumPostReward / (post3Weight + post4Weight)
	post4Reward := post4Weight * sumPostReward / (post3Weight + post4Weight)
	sumPostDappReward := (post3Weight + post4Weight) * (globalPostDappReward + perBlockPostDappReward(d)) / exceptNextBlockPostWeightedVps
	post3DappReward := post3Weight * sumPostDappReward / (post3Weight + post4Weight)
	post4DappReward := post4Weight * sumPostDappReward / (post3Weight + post4Weight)
	reward3 := post3Reward + post3DappReward
	reward4 := post4Reward + post4DappReward

	a.NoError(d.ProduceBlocks(1))
	a.Equal(d.GlobalProps().GetPostWeightedVps(), strconv.FormatUint(exceptNextBlockPostWeightedVps, 10))
	vestnew3 := d.Account(tester.acc0.Name).GetVest().Value
	vestnew4 := d.Account(tester.acc1.Name).GetVest().Value
	real3Reward := vestnew3 - vestold3
	real4Reward := vestnew4 - vestold4
	a.Equal(reward3, real3Reward)
	a.Equal(reward4, real4Reward)
}
