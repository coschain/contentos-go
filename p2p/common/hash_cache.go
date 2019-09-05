package common

import (
	mapset "github.com/deckarep/golang-set"
	"sync"
)

const (
	HashSize = 32
	HashCacheMaxCount = 32768
)

type HashCache struct {
	messageCount     int
	filter1          mapset.Set
	filter2          mapset.Set
	useFilter2       bool
	sync.RWMutex
}

func NewHashCache() *HashCache {
	return &HashCache{
		filter1:      mapset.NewSet(),
		filter2:      mapset.NewSet(),
	}
}

func (c *HashCache) has(hash [HashSize]byte) bool {
	if c.useFilter2 {
		return c.filter2.Contains(hash)
	}
	return c.filter1.Contains(hash)
}

func (c *HashCache) put(hash [HashSize]byte) {
	c.messageCount++
	if c.messageCount <= HashCacheMaxCount / 2 {
		c.filter1.Add(hash)
	} else {
		c.filter1.Add(hash)
		c.filter2.Add(hash)
		if c.messageCount == HashCacheMaxCount {
			if c.useFilter2 {
				c.useFilter2 = false
				c.filter2 = mapset.NewSet()
			} else {
				c.useFilter2 = true
				c.filter1 = mapset.NewSet()
			}
			c.messageCount = HashCacheMaxCount / 2
		}
	}
}

func (c *HashCache) Has(hash [HashSize]byte) bool {
	c.RLock()
	defer c.RUnlock()

	return c.has(hash)
}

func (c *HashCache) Put(hash [HashSize]byte) {
	c.Lock()
	defer c.Unlock()

	c.put(hash)
}

func (c *HashCache) PutIfNotFound(hash [HashSize]byte) (changed bool) {
	c.Lock()
	defer c.Unlock()

	changed = !c.has(hash)
	if changed {
		c.put(hash)
	}
	return
}
