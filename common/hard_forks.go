package common

const (
	Original uint64 = iota
)

var hardForks = []uint64{
	Original,
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
