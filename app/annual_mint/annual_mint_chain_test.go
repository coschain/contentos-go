package annual_mint

import (
	"github.com/coschain/contentos-go/common/constants"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBudget(t *testing.T) {
	var mintValue = [][]uint64{
		{ 1, 44800000},
		{ 2, 89600000},
		{ 3, 134400000},
		{ 4, 179200000},
		{ 5, 224000000},
		{ 6, 268800000},
		{ 7, 313600000},
		{ 8, 358400000},
		{ 9, 403200000},
		{ 10,448000000},
		{ 11,492800000},
		{ 12,543200000},
	}
	a := assert.New(t)

	var totalValue uint64 = 0
	for i:= 0; i < len(mintValue); i++ {
		a.Equal( mintValue[i][1] , CalculateBudget( uint32(mintValue[i][0]) ) / constants.COSTokenDecimals )
		totalValue += mintValue[i][1]
	}

	a.Equal(totalValue, uint64(3500000000) )
}
