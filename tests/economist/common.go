package economist

import (
	"github.com/coschain/contentos-go/app/annual_mint"
	"github.com/coschain/contentos-go/common/constants"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

var mintProps = &prototype.ChainProperties{
	AccountCreationFee: prototype.NewCoin(1),
	StaminaFree:        constants.DefaultStaminaFree,
	TpsExpected:        constants.DefaultTPSExpected,
	EpochDuration:      constants.InitEpochDuration,
	TopNAcquireFreeToken: constants.InitTopN,
	PerTicketPrice:     prototype.NewCoin(1000000),
	PerTicketWeight:    constants.PerTicketWeight,
}

func registerBlockProducer(account *DandelionAccount, t *testing.T)  {
	a := assert.New(t)
	a.NoError(account.SendTrxAndProduceBlock(TransferToVest(account.Name, account.Name, constants.MinBpRegisterVest, "")))
	a.NoError(account.SendTrxAndProduceBlock(BpRegister(account.Name, "", "", account.GetPubKey(), mintProps)))
}

func RegisterBlockProducer(account *DandelionAccount, t *testing.T)  {
	registerBlockProducer(account, t)
}

func SelfTransferToVesting(accounts []*DandelionAccount, amount uint64, t *testing.T) {
	a := assert.New(t)
	for _, account := range accounts {
		a.NoError(account.SendTrxAndProduceBlock(TransferToVest(account.Name, account.Name, amount, "")))
	}
}

func bigDecay(rawValue *big.Int) *big.Int {
	var decayValue big.Int
	decayValue.Mul(rawValue, new(big.Int).SetUint64(constants.BlockInterval))
	decayValue.Div(&decayValue, new(big.Int).SetUint64(constants.VpDecayTime))
	rawValue.Sub(rawValue, &decayValue)
	return rawValue
}

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

func perBlockReplyReward(d *Dandelion) uint64 {
	ith := d.GlobalProps().GetIthYear()
	annualBudget := annual_mint.CalculateBudget(ith)
	blockCurrency := annual_mint.CalculatePerBlockBudget(annualBudget)

	creatorReward := blockCurrency * constants.RewardRateCreator / constants.PERCENT
	replyReward := creatorReward * constants.RewardRateReply / constants.PERCENT
	return replyReward
}

func perBlockDappReward(d *Dandelion) uint64 {
	ith := d.GlobalProps().GetIthYear()
	annualBudget := annual_mint.CalculateBudget(ith)
	blockCurrency := annual_mint.CalculatePerBlockBudget(annualBudget)

	dappReward := blockCurrency * constants.RewardRateDapp / constants.PERCENT
	return dappReward
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