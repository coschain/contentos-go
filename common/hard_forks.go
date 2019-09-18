package common

const (
	Original uint64 = iota
	HardFork1 = 1458000
)

var hardForks = []uint64{
	Original,
	HardFork1,
}

func GetHardFork(blockNum uint64) uint64 {
	r := Original
	for _, hf := range hardForks {
		if blockNum >= hf {
			r = hf
		}
	}
	return r
}
