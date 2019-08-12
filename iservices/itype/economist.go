package itype

import (
	"github.com/coschain/contentos-go/prototype"
	"math/big"
)

type VoteProxy struct {
	VoteId *prototype.VoterId
	WeightedVp *big.Int
}
