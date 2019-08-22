package economist

import (
	"github.com/coschain/contentos-go/app"
	"github.com/coschain/contentos-go/common/constants"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/stretchr/testify/assert"
	"math"
	"math/big"
	"testing"
)

type UtilTester struct {}

func (tester *UtilTester) Test(t *testing.T, d *Dandelion) {
	proportionAlgorithmTest(t)
	decay(t)
	equalZero(t)
}

func proportionAlgorithmTest(t *testing.T) {
	a := assert.New(t)
	result1 := app.ProportionAlgorithm(new(big.Int).SetUint64(1), new(big.Int).SetUint64(constants.VpDecayTime), new(big.Int).SetUint64(1000)).Uint64()
	result2 := uint64(1) * uint64(1000) / uint64(constants.VpDecayTime)
	a.Equal(result1, result2)
	result3 := app.ProportionAlgorithm(new(big.Int).SetUint64(1), new(big.Int).SetUint64(0), new(big.Int).SetUint64(1000)).Uint64()
	a.Equal(uint64(0), result3)
	result4 := app.ProportionAlgorithm(new(big.Int).SetUint64(0), new(big.Int).SetUint64(1000), new(big.Int).SetUint64(1000)).Uint64()
	a.Equal(uint64(0), result4)
	result5 := app.ProportionAlgorithm(new(big.Int).SetUint64(1), new(big.Int).SetUint64(1000), new(big.Int).SetUint64(0)).Uint64()
	a.Equal(uint64(0), result5)
	result6 := app.ProportionAlgorithm(new(big.Int).SetUint64(5), new(big.Int).SetUint64(math.MaxUint64), new(big.Int).SetUint64(math.MaxUint64)).Uint64()
	a.Equal(uint64(5), result6)
}

func decay(t *testing.T) {
	a := assert.New(t)
	result1 := app.Decay(new(big.Int).SetUint64(constants.VpDecayTime)).Uint64()
	a.Equal(result1, uint64(constants.VpDecayTime) - uint64(1))
	result2 := app.Decay(new(big.Int).SetUint64(constants.VpDecayTime - 1)).Uint64()
	a.Equal(result2, uint64(constants.VpDecayTime - 1))
	result3 :=  app.Decay(new(big.Int).SetUint64(0)).Uint64()
	a.Equal(result3, uint64(0))
	beforeDecay := new(big.Int).SetUint64(constants.VpDecayTime)
	app.Decay(beforeDecay)
	a.Equal(beforeDecay.Uint64(), uint64(constants.VpDecayTime) - uint64(1))
}

func equalZero(t *testing.T) {
	a := assert.New(t)
	a.True(app.EqualZero(new(big.Int).SetUint64(0)))
	a.True(app.EqualZero(new(big.Int).SetInt64(0)))
	a.False(app.EqualZero(new(big.Int).SetUint64(1)))
}

//func greaterThanZero(t *testing.T) {
//	a := assert.New(t)
//	a.True(app.GreaterThanZero(new(big.Int).SetUint64(0)))
//	a.True(app.EqualZero(new(big.Int).SetInt64(0)))
//	a.False(app.EqualZero(new(big.Int).SetUint64(1)))
//}