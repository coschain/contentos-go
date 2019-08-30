// +build tests

package annual_mint

import "github.com/coschain/contentos-go/common/constants"

func BaseBudget(ith uint32) uint64 {
	if ith > 12 {
		return 0
	}
	var remain uint64 = 0
	if ith == 12 {
		remain = uint64(constants.COSTotalSupply)  / 1000 / 100 * uint64(56)
	}
	return uint64(constants.COSTotalSupply)  / 1000 / 100 * uint64(448) * uint64(ith) + remain
}


// InitialBonus does not be managed by chain
func CalculateBudget(ith uint32) uint64 {
	return BaseBudget(ith)
}

// let 8640 block to one year
func CalculatePerBlockBudget(annalBudget uint64) uint64 {
	return annalBudget / (8640 / constants.BlockInterval)
}
