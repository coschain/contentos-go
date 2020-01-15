package op

import (
	"github.com/coschain/contentos-go/common/constants"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

type ConvertVestTester struct {
	acc0, acc1, acc2, acc3, acc4, acc5 *DandelionAccount
}

var cvProps = &prototype.ChainProperties{
	AccountCreationFee: prototype.NewCoin(1),
	StaminaFree:        constants.DefaultStaminaFree,
	TpsExpected:        constants.DefaultTPSExpected,
	EpochDuration:      constants.InitEpochDuration,
	TopNAcquireFreeToken: constants.InitTopN,
	PerTicketPrice:     prototype.NewCoin(1000000),
	PerTicketWeight:    constants.PerTicketWeight,
}

func (tester *ConvertVestTester) TestHardFork2DoNothing(t *testing.T, d *Dandelion) {
	tester.acc2 = d.Account("actor2")
	tester.acc5 = d.Account("actor5")

	a := assert.New(t)
	a.NoError(tester.acc2.SendTrxAndProduceBlock(TransferToVest(tester.acc2.Name, tester.acc2.Name, constants.MinBpRegisterVest, "")))
	a.NoError(tester.acc2.SendTrxAndProduceBlock(BpRegister(tester.acc2.Name, "", "", tester.acc2.GetPubKey(), cvProps)))


	// 1. before HardFork2 power down
	// 2. HardFork2 happen
	// 3. do nothing
	t.Run("HardFork2 do nothing", d.Test(tester.hf2DoNothing))
}

func (tester *ConvertVestTester) TestHardFork2Clear(t *testing.T, d *Dandelion) {
	tester.acc2 = d.Account("actor2")
	tester.acc5 = d.Account("actor5")

	a := assert.New(t)
	a.NoError(tester.acc2.SendTrxAndProduceBlock(TransferToVest(tester.acc2.Name, tester.acc2.Name, constants.MinBpRegisterVest, "")))
	a.NoError(tester.acc2.SendTrxAndProduceBlock(BpRegister(tester.acc2.Name, "", "", tester.acc2.GetPubKey(), cvProps)))


	// 1. before HardFork2 power down
	// 2. HardFork2 happen
	// 3. start a new power down
	t.Run("HardFork2 interrupt former power down", d.Test(tester.hf2ClearFormerBehave))
}

func (tester *ConvertVestTester) Test(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")
	tester.acc3 = d.Account("actor3")
	tester.acc4 = d.Account("actor4")

	a := assert.New(t)
	a.NoError(tester.acc2.SendTrxAndProduceBlock(TransferToVest(tester.acc2.Name, tester.acc2.Name, constants.MinBpRegisterVest, "")))
	a.NoError(tester.acc2.SendTrxAndProduceBlock(BpRegister(tester.acc2.Name, "", "", tester.acc2.GetPubKey(), cvProps)))

	t.Run("normal", d.Test(tester.normal))
	t.Run("reset", d.Test(tester.Reset))
	t.Run("too much", d.Test(tester.tooMuch))
	t.Run("too small", d.Test(tester.tooSmall))
	t.Run("mismatch", d.Test(tester.mismatch))
}

func (tester *ConvertVestTester) normal(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const TRANSFER = 10000000
	a.NoError(tester.acc0.SendTrxAndProduceBlock(TransferToVest(tester.acc0.Name, tester.acc0.Name, TRANSFER, "")))

	vest0 := d.Account(tester.acc0.Name).GetVest().Value
	balance0 := d.Account(tester.acc0.Name).GetBalance().Value

	headBlock0 := d.GlobalProps().GetHeadBlockNumber()
	a.Equal(uint64(0), d.Account(tester.acc0.Name).GetStartPowerdownBlockNum())
	a.NoError(tester.acc0.SendTrxAndProduceBlock(ConvertVest(tester.acc0.Name, TRANSFER)))
	a.Equal(headBlock0 + uint64(constants.PowerDownBlockInterval), d.Account(tester.acc0.Name).GetNextPowerdownBlockNum())
	a.Equal(headBlock0, d.Account(tester.acc0.Name).GetStartPowerdownBlockNum())
	eachRate := TRANSFER / (constants.ConvertWeeks - 1)
	a.Equal(uint64(eachRate), d.Account(tester.acc0.Name).GetEachPowerdownRate().Value)
	a.NoError(d.ProduceBlocks(constants.PowerDownBlockInterval + 1))
	a.Equal(balance0 + uint64(eachRate), d.Account(tester.acc0.Name).GetBalance().Value)
	a.Equal(headBlock0, d.Account(tester.acc0.Name).GetStartPowerdownBlockNum())
	a.NoError(d.ProduceBlocks(constants.PowerDownBlockInterval + 1))
	a.Equal(balance0 + uint64(eachRate) * 2, d.Account(tester.acc0.Name).GetBalance().Value)
	a.Equal(headBlock0, d.Account(tester.acc0.Name).GetStartPowerdownBlockNum())
	a.NoError(d.ProduceBlocks(constants.PowerDownBlockInterval * 20))
	a.Equal(balance0 + TRANSFER, d.Account(tester.acc0.Name).GetBalance().Value)
	a.Equal(vest0 - TRANSFER, d.Account(tester.acc0.Name).GetVest().Value)
	a.Equal(uint64(0), d.Account(tester.acc0.Name).GetEachPowerdownRate().Value)
	a.Equal(uint64(math.MaxUint64), d.Account(tester.acc0.Name).GetNextPowerdownBlockNum())
	a.Equal(uint64(0), d.Account(tester.acc0.Name).GetStartPowerdownBlockNum())

	// apply hardfork2
	a.NoError(d.ProduceBlocks(int(constants.HardFork2)))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(TransferToVest(tester.acc0.Name, tester.acc0.Name, TRANSFER, "")))
	headBlock1 := d.GlobalProps().GetHeadBlockNumber()
	balance1 := d.Account(tester.acc0.Name).GetBalance().Value
	vest1 := d.Account(tester.acc0.Name).GetVest().Value
	a.NoError(tester.acc0.SendTrxAndProduceBlock(ConvertVest(tester.acc0.Name, TRANSFER)))
	a.Equal(headBlock1 + uint64(constants.PowerDownBlockInterval), d.Account(tester.acc0.Name).GetNextPowerdownBlockNum())
	a.Equal(headBlock1, d.Account(tester.acc0.Name).GetStartPowerdownBlockNum())
	eachRate = TRANSFER / (constants.HardFork2ConvertWeeks - 1)
	a.Equal(uint64(eachRate), d.Account(tester.acc0.Name).GetEachPowerdownRate().Value)
	a.NoError(d.ProduceBlocks(constants.PowerDownBlockInterval + 1))
	a.Equal(balance1 + uint64(eachRate), d.Account(tester.acc0.Name).GetBalance().Value)

	// produce blocks to finish power down
	a.NoError(d.ProduceBlocks(constants.HardFork2ConvertWeeks * constants.PowerDownBlockInterval + 1))
	balance2 := d.Account(tester.acc0.Name).GetBalance().Value
	vest2 := d.Account(tester.acc0.Name).GetVest().Value
	a.Equal(vest1 - vest2, uint64(TRANSFER))
	a.Equal(balance2 - balance1, uint64(TRANSFER))
}

func (tester *ConvertVestTester) Reset(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const (
		TRANSFER = 10000000
		TRANSFER2 = 2000000
	)

	a.NoError(tester.acc1.SendTrxAndProduceBlock(TransferToVest(tester.acc1.Name, tester.acc1.Name, TRANSFER, "")))

	vest1 := d.Account(tester.acc1.Name).GetVest().Value
	balance1 := d.Account(tester.acc1.Name).GetBalance().Value
	headBlock1 := d.GlobalProps().GetHeadBlockNumber()
	a.Equal(uint64(0), d.Account(tester.acc1.Name).GetStartPowerdownBlockNum())

	a.NoError(tester.acc1.SendTrxAndProduceBlock(ConvertVest(tester.acc1.Name, TRANSFER)))
	a.Equal(headBlock1 + uint64(constants.PowerDownBlockInterval), d.Account(tester.acc1.Name).GetNextPowerdownBlockNum())
	a.Equal(headBlock1, d.Account(tester.acc1.Name).GetStartPowerdownBlockNum())
	eachRate1 := TRANSFER / (constants.HardFork2ConvertWeeks - 1)
	a.Equal(uint64(eachRate1), d.Account(tester.acc1.Name).GetEachPowerdownRate().Value)
	a.NoError(d.ProduceBlocks(constants.PowerDownBlockInterval + 1))
	a.Equal(headBlock1, d.Account(tester.acc1.Name).GetStartPowerdownBlockNum())
	a.Equal(balance1 + uint64(eachRate1), d.Account(tester.acc1.Name).GetBalance().Value)
	a.Equal(vest1- uint64(eachRate1), d.Account(tester.acc1.Name).GetVest().Value)

	vest2 := d.Account(tester.acc1.Name).GetVest().Value
	balance2 := d.Account(tester.acc1.Name).GetBalance().Value
	headBlock2 := d.GlobalProps().GetHeadBlockNumber()

	a.NoError(tester.acc1.SendTrxAndProduceBlock(ConvertVest(tester.acc1.Name, TRANSFER2)))
	a.Equal(headBlock2 + uint64(constants.PowerDownBlockInterval), d.Account(tester.acc1.Name).GetNextPowerdownBlockNum())
	eachRate2 := TRANSFER2 / (constants.HardFork2ConvertWeeks - 1)
	a.Equal(uint64(eachRate2), d.Account(tester.acc1.Name).GetEachPowerdownRate().Value)
	a.Equal(headBlock2, d.Account(tester.acc1.Name).GetStartPowerdownBlockNum())
	a.NoError(d.ProduceBlocks(constants.PowerDownBlockInterval + 1))
	a.Equal(balance2 + uint64(eachRate2), d.Account(tester.acc1.Name).GetBalance().Value)
	a.Equal(vest2 - uint64(eachRate2), d.Account(tester.acc1.Name).GetVest().Value)
	a.Equal(headBlock2, d.Account(tester.acc1.Name).GetStartPowerdownBlockNum())
}

func (tester *ConvertVestTester) tooMuch(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	const TRANSFER = 10000000
	a.NoError(tester.acc3.SendTrxAndProduceBlock(TransferToVest(tester.acc3.Name, tester.acc3.Name, TRANSFER, "")))
	vest := d.Account(tester.acc3.Name).GetVest().Value
	toConvert := d.Account(tester.acc3.Name).GetToPowerdown().Value

	a.NoError(tester.acc3.SendTrx(ConvertVest(tester.acc3.Name, vest+ 1)))
	a.NoError(d.ProduceBlocks(1))
	a.NoError(tester.acc3.SendTrx(ConvertVest(tester.acc3.Name, math.MaxUint64)))
	a.NoError(d.ProduceBlocks(1))
	a.Equal(vest, d.Account(tester.acc3.Name).GetVest().Value)
	a.Equal(toConvert, d.Account(tester.acc3.Name).GetToPowerdown().Value)
}

func (tester *ConvertVestTester) tooSmall(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	const TRANSFER = 10000000
	a.NoError(tester.acc4.SendTrxAndProduceBlock(TransferToVest(tester.acc4.Name, tester.acc4.Name, TRANSFER, "")))
	vest := d.Account(tester.acc4.Name).GetVest().Value
	toConvert := d.Account(tester.acc4.Name).GetToPowerdown().Value
	a.Error(tester.acc4.SendTrx(ConvertVest(tester.acc4.Name, 1)))
	a.NoError(d.ProduceBlocks(1))
	a.Equal(vest, d.Account(tester.acc4.Name).GetVest().Value)
	a.Equal(toConvert, d.Account(tester.acc4.Name).GetToPowerdown().Value)
}

func (tester *ConvertVestTester) mismatch(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	const TRANSFER = 10000000
	vest := d.Account(tester.acc0.Name).GetVest().Value
	toConvert := d.Account(tester.acc0.Name).GetToPowerdown().Value
	a.Error(tester.acc0.SendTrx(ConvertVest(tester.acc1.Name, TRANSFER)))
	a.NoError(d.ProduceBlocks(1))
	a.Equal(vest, d.Account(tester.acc0.Name).GetVest().Value)
	a.Equal(toConvert, d.Account(tester.acc0.Name).GetToPowerdown().Value)
}

func (tester *ConvertVestTester) hf2ClearFormerBehave(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const TRANSFER = 10000000
	a.NoError(tester.acc5.SendTrxAndProduceBlock(TransferToVest(tester.acc5.Name, tester.acc5.Name, 2 * TRANSFER, "")))

	// produce some blocks to get close to HardFork2 point
	a.NoError(d.ProduceBlocks( int(constants.HardFork2 - constants.PowerDownBlockInterval - 5) ))
	vest0 := d.Account(tester.acc5.Name).GetVest().Value
	balance0 := d.Account(tester.acc5.Name).GetBalance().Value
	headBlock0 := d.GlobalProps().GetHeadBlockNumber()
	a.Equal(uint64(0), d.Account(tester.acc5.Name).GetStartPowerdownBlockNum())

	// start a power down before HardFork2, it should follow old rule
	a.NoError(tester.acc5.SendTrxAndProduceBlock(ConvertVest(tester.acc5.Name, TRANSFER)))
	a.Equal(headBlock0 + uint64(constants.PowerDownBlockInterval), d.Account(tester.acc5.Name).GetNextPowerdownBlockNum())
	a.Equal(headBlock0, d.Account(tester.acc5.Name).GetStartPowerdownBlockNum())
	eachRate := TRANSFER / (constants.ConvertWeeks - 1)
	a.Equal(uint64(eachRate), d.Account(tester.acc5.Name).GetEachPowerdownRate().Value)
	a.NoError(d.ProduceBlocks(constants.PowerDownBlockInterval + 1))
	a.Equal(balance0 + uint64(eachRate), d.Account(tester.acc5.Name).GetBalance().Value)
	a.Equal(vest0 - uint64(eachRate), d.Account(tester.acc5.Name).GetVest().Value)

	// produce some blocks to more than HardFork2 but former power down not yet done
	a.NoError(d.ProduceBlocks( constants.PowerDownBlockInterval ))

	// start a new power down, it should follow new rule
	vest1 := d.Account(tester.acc5.Name).GetVest().Value
	balance1 := d.Account(tester.acc5.Name).GetBalance().Value
	a.NoError(tester.acc5.SendTrxAndProduceBlock(ConvertVest(tester.acc5.Name, TRANSFER)))
	eachRate = TRANSFER / (constants.HardFork2ConvertWeeks - 1)
	a.Equal(uint64(eachRate), d.Account(tester.acc5.Name).GetEachPowerdownRate().Value)
	a.NoError(d.ProduceBlocks(constants.PowerDownBlockInterval + 1))
	a.Equal(balance1 + uint64(eachRate), d.Account(tester.acc5.Name).GetBalance().Value)
	a.Equal(vest1 - uint64(eachRate), d.Account(tester.acc5.Name).GetVest().Value)

	// produce blocks to finish power down
	a.NoError(d.ProduceBlocks(constants.HardFork2ConvertWeeks * constants.PowerDownBlockInterval + 1))
	balance2 := d.Account(tester.acc5.Name).GetBalance().Value
	vest2 := d.Account(tester.acc5.Name).GetVest().Value
	a.Equal(vest1 - vest2, uint64(TRANSFER))
	a.Equal(balance2 - balance1, uint64(TRANSFER))
}

func (tester *ConvertVestTester) hf2DoNothing(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const TRANSFER = 10000000
	a.NoError(tester.acc5.SendTrxAndProduceBlock(TransferToVest(tester.acc5.Name, tester.acc5.Name, TRANSFER, "")))

	// produce some blocks to get close to HardFork2 point
	a.NoError(d.ProduceBlocks( int(constants.HardFork2 - constants.PowerDownBlockInterval - 5) ))
	vest0 := d.Account(tester.acc5.Name).GetVest().Value
	balance0 := d.Account(tester.acc5.Name).GetBalance().Value
	headBlock0 := d.GlobalProps().GetHeadBlockNumber()
	a.Equal(uint64(0), d.Account(tester.acc5.Name).GetStartPowerdownBlockNum())

	// start a power down before HardFork2, it should follow old rule
	a.NoError(tester.acc5.SendTrxAndProduceBlock(ConvertVest(tester.acc5.Name, TRANSFER)))
	a.Equal(headBlock0 + uint64(constants.PowerDownBlockInterval), d.Account(tester.acc5.Name).GetNextPowerdownBlockNum())
	a.Equal(headBlock0, d.Account(tester.acc5.Name).GetStartPowerdownBlockNum())
	eachRate := TRANSFER / (constants.ConvertWeeks - 1)
	a.Equal(uint64(eachRate), d.Account(tester.acc5.Name).GetEachPowerdownRate().Value)
	a.NoError(d.ProduceBlocks(constants.PowerDownBlockInterval + 1))
	a.Equal(balance0 + uint64(eachRate), d.Account(tester.acc5.Name).GetBalance().Value)
	a.Equal(vest0 - uint64(eachRate), d.Account(tester.acc5.Name).GetVest().Value)

	// produce some blocks to more than HardFork2 but former power down not yet done
	a.NoError(d.ProduceBlocks( constants.PowerDownBlockInterval ))
	eachRateCorrect := TRANSFER / (constants.ConvertWeeks - 1)
	eachRateWrong := TRANSFER / (constants.HardFork2ConvertWeeks - 1)
	a.Equal(uint64(eachRateCorrect), d.Account(tester.acc5.Name).GetEachPowerdownRate().Value)
	a.NotEqual(uint64(eachRateWrong), d.Account(tester.acc5.Name).GetEachPowerdownRate().Value)

	// produce HardFork2ConvertWeeks blocks, power down should't be done
	a.NoError(d.ProduceBlocks(constants.HardFork2ConvertWeeks * constants.PowerDownBlockInterval + 1))
	balance1 := d.Account(tester.acc5.Name).GetBalance().Value
	vest1 := d.Account(tester.acc5.Name).GetVest().Value
	a.NotEqual(vest0 - vest1, uint64(TRANSFER))
	a.NotEqual(balance1 - balance0, uint64(TRANSFER))

	// continue produce HardFork2ConvertWeeks blocks, power down should be done
	a.NoError(d.ProduceBlocks(constants.HardFork2ConvertWeeks * constants.PowerDownBlockInterval + 1))
	balance2 := d.Account(tester.acc5.Name).GetBalance().Value
	vest2 := d.Account(tester.acc5.Name).GetVest().Value
	a.Equal(vest0 - vest2, uint64(TRANSFER))
	a.Equal(balance2 - balance0, uint64(TRANSFER))
}