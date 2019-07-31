package test

import (
	"github.com/coschain/contentos-go/iservices"
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
	seed := rand.Uint64()
	seed |= rand.Uint64()
	if seed == 0 {
		seed = 1
	}

	val := make(map[string]bool)
	m.RLock()
	for k, v := range m.validators {
		val[k] = v
	}
	m.RUnlock()

	if val[names[0]] == false {
		if err := EnableBP(names[0], sks[0], css); err != nil {
			//panic(err)
		}
	}

	for i := 1; i < size; i++ {
		if (seed>>uint(i)&1) == 1 && val[names[i]] == false {
			if err := EnableBP(names[i], sks[i], css); err != nil {
				//panic(err)
			}
		}
		if (seed>>uint(i)&1) == 0 && val[names[i]] == true {
			if err := DisableBP(names[i], sks[i], css); err != nil {
				//panic(err)
			}
		}
	}
}
