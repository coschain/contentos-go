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
	acc0, acc1, acc2, acc3, acc4 *DandelionAccount
}

var cvProps = &prototype.ChainProperties{
	AccountCreationFee: prototype.NewCoin(1),
	MaximumBlockSize:   1024 * 1024,
	StaminaFree:        constants.DefaultStaminaFree,
	TpsExpected:        constants.DefaultTPSExpected,
	EpochDuration:      constants.InitEpochDuration,
	TopNAcquireFreeToken: constants.InitTopN,
	PerTicketPrice:     prototype.NewCoin(1000000),
	PerTicketWeight:    constants.PerTicketWeight,
}

func (tester *ConvertVestTester) Test(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")
	tester.acc3 = d.Account("actor3")
	tester.acc4 = d.Account("actor4")

	a := assert.New(t)
	a.NoError(tester.acc2.SendTrxAndProduceBlock(TransferToVest(tester.acc2.Name, tester.acc2.Name, constants.MinBpRegisterVest)))
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
	a.NoError(tester.acc0.SendTrxAndProduceBlock(TransferToVest(tester.acc0.Name, tester.acc0.Name, TRANSFER)))

	vest0 := d.Account(tester.acc0.Name).GetVest().Value
	balance0 := d.Account(tester.acc0.Name).GetBalance().Value

	headBlock0 := d.GlobalProps().GetHeadBlockNumber()
	a.NoError(tester.acc0.SendTrxAndProduceBlock(ConvertVest(tester.acc0.Name, TRANSFER)))
	a.Equal(headBlock0 + uint64(constants.PowerDownBlockInterval), d.Account(tester.acc0.Name).GetNextPowerdownBlockNum())
	eachRate := TRANSFER / (constants.ConvertWeeks - 1)
	a.Equal(uint64(eachRate), d.Account(tester.acc0.Name).GetEachPowerdownRate().Value)
	a.NoError(d.ProduceBlocks(constants.PowerDownBlockInterval + 1))
	a.Equal(balance0 + uint64(eachRate), d.Account(tester.acc0.Name).GetBalance().Value)
	a.NoError(d.ProduceBlocks(constants.PowerDownBlockInterval + 1))
	a.Equal(balance0 + uint64(eachRate) * 2, d.Account(tester.acc0.Name).GetBalance().Value)
	a.NoError(d.ProduceBlocks(constants.PowerDownBlockInterval * 20))
	a.Equal(balance0 + TRANSFER, d.Account(tester.acc0.Name).GetBalance().Value)
	a.Equal(vest0 - TRANSFER, d.Account(tester.acc0.Name).GetVest().Value)
	a.Equal(uint64(0), d.Account(tester.acc0.Name).GetEachPowerdownRate().Value)
	a.Equal(uint64(math.MaxUint64), d.Account(tester.acc0.Name).GetNextPowerdownBlockNum())
}

func (tester *ConvertVestTester) Reset(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const (
		TRANSFER = 10000000
		TRANSFER2 = 2000000
	)

	a.NoError(tester.acc1.SendTrxAndProduceBlock(TransferToVest(tester.acc1.Name, tester.acc1.Name, TRANSFER)))

	vest1 := d.Account(tester.acc1.Name).GetVest().Value
	balance1 := d.Account(tester.acc1.Name).GetBalance().Value
	headBlock1 := d.GlobalProps().GetHeadBlockNumber()

	a.NoError(tester.acc1.SendTrxAndProduceBlock(ConvertVest(tester.acc1.Name, TRANSFER)))
	a.Equal(headBlock1 + uint64(constants.PowerDownBlockInterval), d.Account(tester.acc1.Name).GetNextPowerdownBlockNum())
	eachRate1 := TRANSFER / (constants.ConvertWeeks - 1)
	a.Equal(uint64(eachRate1), d.Account(tester.acc1.Name).GetEachPowerdownRate().Value)
	a.NoError(d.ProduceBlocks(constants.PowerDownBlockInterval + 1))
	a.Equal(balance1 + uint64(eachRate1), d.Account(tester.acc1.Name).GetBalance().Value)
	a.Equal(vest1- uint64(eachRate1), d.Account(tester.acc1.Name).GetVest().Value)

	vest2 := d.Account(tester.acc1.Name).GetVest().Value
	balance2 := d.Account(tester.acc1.Name).GetBalance().Value
	headBlock2 := d.GlobalProps().GetHeadBlockNumber()

	a.NoError(tester.acc1.SendTrxAndProduceBlock(ConvertVest(tester.acc1.Name, TRANSFER2)))
	a.Equal(headBlock2 + uint64(constants.PowerDownBlockInterval), d.Account(tester.acc1.Name).GetNextPowerdownBlockNum())
	eachRate2 := TRANSFER2 / (constants.ConvertWeeks - 1)
	a.Equal(uint64(eachRate2), d.Account(tester.acc1.Name).GetEachPowerdownRate().Value)
	a.NoError(d.ProduceBlocks(constants.PowerDownBlockInterval + 1))
	a.Equal(balance2 + uint64(eachRate2), d.Account(tester.acc1.Name).GetBalance().Value)
	a.Equal(vest2 - uint64(eachRate2), d.Account(tester.acc1.Name).GetVest().Value)
}

func (tester *ConvertVestTester) tooMuch(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	const TRANSFER = 10000000
	a.NoError(tester.acc3.SendTrxAndProduceBlock(TransferToVest(tester.acc3.Name, tester.acc3.Name, TRANSFER)))
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
	a.NoError(tester.acc4.SendTrxAndProduceBlock(TransferToVest(tester.acc4.Name, tester.acc4.Name, TRANSFER)))
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