package variables

import (
	"strconv"
)

var (
	POSTCASHOUTDELAYBLOCK = "86400"
	VPDECAYTIME = "129600"
)

func VpDecayTime() uint64 {
	v, err := strconv.ParseUint(VPDECAYTIME, 10, 64)
	if err != nil {
		return 129600
	}
	return v
}

func PostCashOutDelayBlock() uint64 {
	p, err := strconv.ParseUint(POSTCASHOUTDELAYBLOCK, 10, 64)
	if err != nil {
		return 86400
	}
	return p
}
