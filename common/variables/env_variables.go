package variables

import (
	"os"
	"strconv"
)

func PostCashOutDelayBlock() uint64 {
	var delayBlock uint64
	delayBlockStr := os.Getenv("PostCashOutDelayBlock")
	if len(delayBlockStr) == 0 {
		delayBlock = 60 * 60 * 24
	}
	delayBlock, err := strconv.ParseUint(delayBlockStr, 10, 64)
	if err != nil {
		delayBlock = 60 * 60 * 24
	}
	return delayBlock
}

func VpDecayTime() uint64 {
	var vpDecayTime uint64
	vpDecayTimeStr := os.Getenv("VpDecayTime")
	if len(vpDecayTimeStr) == 0 {
		vpDecayTime = 60 * 60 * 24 * 1.5
	}
	vpDecayTime, err := strconv.ParseUint(vpDecayTimeStr, 10, 64)
	if err != nil {
		vpDecayTime= 60 * 60 * 24 * 1.5
	}
	return vpDecayTime
}
