package op

import (
	"fmt"
	"github.com/coschain/contentos-go/common/constants"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/stretchr/testify/assert"
	"testing"
)

type ContractGasTester struct {
	seed uint32
	cpu uint64
}

func NewContractGasTester(seed uint32, cpu uint64) *ContractGasTester {
	return &ContractGasTester{ seed:seed, cpu:cpu }
}

func NewContractGasTest(seed uint32, cpu uint64) func(*testing.T) {
	return NewDandelionContractTest(NewContractGasTester(seed, cpu).Test, 0, 1, "actor0.gas_burner")
}

func (tester *ContractGasTester) Test(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	// first, initminer stakes 80% of coins for actor0
	initminer := d.Account(constants.COSInitMiner)
	a.NoError(initminer.SendTrxAndProduceBlock(Stake(initminer.Name, "actor0", initminer.GetBalance().Value * 4 / 5)))

	// try the same call several times
	for i := 0; i < 5; i++ {
		t.Run(fmt.Sprintf("round_%d", i + 1), d.Test(tester.check))
	}
}

func (tester *ContractGasTester) check(t *testing.T, d *Dandelion) {
	ApplyGas(t, d, tester.cpu, fmt.Sprintf("actor0: actor0.gas_burner.burn %d", tester.seed))
}
