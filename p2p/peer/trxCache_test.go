package peer

import (
	"fmt"
	"github.com/willf/bloom"
	"testing"
	"github.com/magiconair/properties/assert"
)

const (
	maxTrxCount = 10
	bitSize     = 100
	hashFuncnum = 5
)

func TestTrxCache(t *testing.T) {
	fmt.Println("enter trxCache test")
	p := newPeer()

	assert.Equal(t, p.TrxCache.useFilter2, false)
	fmt.Println("init useFilter2 set ok")

	baseStr := "test%d"
	for i:=0;i<10;i++ {
		str := fmt.Sprintf(baseStr, i)

		hash := []byte(str)
		if !p.hasTrx(hash) {
			p.recordTrxCache(hash)
		}
	}

	assert.Equal(t, p.TrxCache.trxCount, maxTrxCount/2)
	fmt.Println("trxCount num correct")

	assert.Equal(t, p.TrxCache.useFilter2, true)
	fmt.Println("useFilter2 change ok")

	testStr := "test5"
	testBytes := []byte(testStr)
	assert.Equal(t, p.hasTrx(testBytes), true)
	fmt.Println("check result correct")
}

func newPeer() *Peer {
	p := &Peer{}

	p.TrxCache.bloomFilter1 = bloom.New(bitSize, hashFuncnum)
	p.TrxCache.bloomFilter2 = bloom.New(bitSize, hashFuncnum)
	p.TrxCache.trxCount = 0
	p.TrxCache.useFilter2 = false

	return p
}

func (this *Peer) hasTrx(hash []byte) bool {
	this.TrxCache.Lock()
	defer this.TrxCache.Unlock()

	if this.TrxCache.useFilter2 == true {
		return this.TrxCache.bloomFilter2.Test(hash)
	}

	return this.TrxCache.bloomFilter1.Test(hash)
}

func (this *Peer) recordTrxCache(hash []byte) {
	this.TrxCache.Lock()
	defer this.TrxCache.Unlock()

	this.TrxCache.trxCount++

	if this.TrxCache.trxCount <= maxTrxCount / 2 {
		this.TrxCache.bloomFilter1.Add(hash)
	} else {
		this.TrxCache.bloomFilter1.Add(hash)
		this.TrxCache.bloomFilter2.Add(hash)

		if this.TrxCache.trxCount == maxTrxCount {
			if this.TrxCache.useFilter2 == true {
				this.TrxCache.useFilter2 = false
				this.TrxCache.bloomFilter2 = bloom.New(bitSize, hashFuncnum)
			} else {
				this.TrxCache.useFilter2 = true
				this.TrxCache.bloomFilter1 = bloom.New(bitSize, hashFuncnum)
			}
			this.TrxCache.trxCount = maxTrxCount / 2
		}
	}
}