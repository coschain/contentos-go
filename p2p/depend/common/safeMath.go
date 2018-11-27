package common

import "math"

const (
	MAX_UINT64 = math.MaxUint64
)

func SafeAdd(x, y uint64) (uint64, bool) {
	return x + y, y > MAX_UINT64-x
}