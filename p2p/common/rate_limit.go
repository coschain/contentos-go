package common

import (
	"sync"
	"time"
)

const (
	MaxBlockQueriesPerSecond      = 1500
	MaxCheckPointQueriesPerSecond = 1500
	MaxBlobSizePerSecond          = 10 * 1024 * 1024
	MaxIncomingConsensusMsgPerSecond = 200
)

const (
	CheckPointSize = 8192	// estimated size of a checkpoint in bytes
)

type RateLimiter struct {
	sync.Mutex
	tokens uint64
	maxTokens uint64
	nanoRecovery uint64
	lastUpdate time.Time
}

const sOneBillion = 1000000000

func NewRateLimiter(limit uint32) *RateLimiter {
	r := new(RateLimiter)
	r.nanoRecovery = uint64(limit)
	r.maxTokens = r.nanoRecovery * sOneBillion
	r.tokens = r.maxTokens
	r.lastUpdate = time.Now()
	return r
}

func (r *RateLimiter) Request(amount uint32, allOrNothing bool) (approved uint32) {
	if r == nil {
		return amount
	}

	r.Lock()
	defer r.Unlock()

	approved = r.request(amount, allOrNothing)
	r.tokens -= uint64(approved) * sOneBillion
	return
}

func (r *RateLimiter) TryRequest(amount uint32, allOrNothing bool) (approved uint32) {
	if r == nil {
		return amount
	}

	r.Lock()
	defer r.Unlock()

	return r.request(amount, allOrNothing)
}

func (r *RateLimiter) request(amount uint32, allOrNothing bool) (approved uint32) {
	r.update()
	if approved = uint32(r.tokens / sOneBillion); approved > amount {
		approved = amount
	}
	if allOrNothing && approved < amount {
		approved = 0
	}
	return
}

func (r *RateLimiter) update() {
	if elapsed := time.Since(r.lastUpdate); elapsed > 0 {
		if elapsed >= time.Second {
			r.tokens = r.maxTokens
		} else if r.tokens < r.maxTokens {
			maxRecovery := r.maxTokens - r.tokens
			recovery := r.nanoRecovery * uint64(elapsed.Nanoseconds())
			if recovery > maxRecovery {
				recovery = maxRecovery
			}
			r.tokens += recovery
		}
		r.lastUpdate = r.lastUpdate.Add(elapsed)
	}
}
