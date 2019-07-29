package test

import (
	"github.com/coschain/contentos-go/iservices"
	"math"
	"math/rand"
	"time"
)

func (m *Monitor) Shuffle(names, sks []string, css iservices.IConsensus, stopCh chan struct{}) {
	ticker := time.NewTicker(5 * time.Second).C
	for {
		select {
		case <-stopCh:
			return
		case <-ticker:
			m.shuffle(names, sks, css)
		}
	}
}

func (m *Monitor) shuffle(names, sks []string, css iservices.IConsensus) {
	size := len(names)
	seed := rand.Uint64() % uint64(math.Pow(2, float64(size-1)))
	if seed == 0 {
		seed = 1
	}

	val := make(map[string]bool)
	m.RLock()
	for k, v := range m.validators {
		val[k] = v
	}
	m.RUnlock()

	for i := 1; i < size; i++ {
		if (seed>>uint(i)&1) == 1 && val[names[i]] == false {
			if err := RegesiterBP(names[i], sks[i], css); err != nil {
				panic(err)
			}
		}
		if (seed>>uint(i)&1) == 0 && val[names[i]] == true {
			if err := UnregesiterBP(names[i], sks[i], css); err != nil {
				panic(err)
			}
		}
	}
}
