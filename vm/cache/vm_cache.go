package vmcache

import (
	"fmt"
	"github.com/go-interpreter/wagon/exec"
	"github.com/hashicorp/golang-lru"
	"sync"
)

const DefaultLruSize = 100

type VmCache struct {
	cache   *lru.Cache						// LRU cache: int64 -> *VM
	counter int64							// auto-incremental counter used as cache key
	byName  map[string]map[int64]bool		// contract -> cache keys
	byIndex map[int64]string				// cache key -> contract
	lock    sync.RWMutex					// for thread safety
}

var once sync.Once
var vc *VmCache

func GetVmCache() *VmCache {
	once.Do(func() {
		vc = &VmCache{
			byName:  make(map[string]map[int64]bool),
			byIndex: make(map[int64]string),
		}
		cache, err := lru.NewWithEvict(DefaultLruSize, vc.onCacheEvict)
		if err != nil {
			panic(err)
		}
		vc.cache = cache
	})
	return vc
}

func buildKey(owner, contract string, codeHash []byte) string {
	const sep = "|"
	return fmt.Sprintf("%s%s%s%s%x", owner, sep, contract, sep, codeHash)
}

// onCacheEvict will be called by lru.Cache for each removed item.
func (v *VmCache) onCacheEvict(key interface{}, value interface{}) {
	idx := key.(int64)
	name := v.byIndex[idx]
	if len(name) > 0 {
		vms := v.byName[name]
		delete(vms, idx)
		if len(vms) == 0 {
			delete(v.byName, name)
		}
		delete(v.byIndex, idx)
	}
}

// Put adds a VM instance to the cache.
func (v *VmCache) Put(owner, contract string, codeHash []byte, vm *exec.VM) {
	v.lock.Lock()
	defer v.lock.Unlock()
	v.counter++
	k, s := v.counter, buildKey(owner, contract, codeHash)
	vms := v.byName[s]
	if vms == nil {
		vms = make(map[int64]bool)
		v.byName[s] = vms
	}
	vms[k] = true
	v.byIndex[k] = s
	v.cache.Add(k, vm)
}

// Fetch fetches a cached VM instance for given contract.
// It returns nil if no cached instance for the contract was found.
func (v *VmCache) Fetch(owner, contract string, codeHash []byte) (vm *exec.VM) {
	v.lock.Lock()
	defer v.lock.Unlock()
	// get all cache keys for the contract
	if vms := v.byName[buildKey(owner, contract, codeHash)]; len(vms) > 0 {
		// pick one key
		var key int64
		for k := range vms {
			key = k
			break
		}
		// get the VM instance from cache using the key, and remove the key from cache.
		if val, ok := v.cache.Peek(key); ok {
			vm = val.(*exec.VM)
			v.cache.Remove(key)
		}
	}
	return
}

// Contains returns number of cached VM instances for the given contract.
func (v *VmCache) Contains(owner, contract string, codeHash []byte) int {
	v.lock.RLock()
	defer v.lock.RUnlock()
	return len(v.byName[buildKey(owner, contract, codeHash)])
}

// Remove deletes all cached VM instances for the given contract.
func (v *VmCache) Remove(owner, contract string, codeHash []byte) {
	v.lock.Lock()
	defer v.lock.Unlock()
	// collect keys that need to delete from the cache
	vms := v.byName[buildKey(owner, contract, codeHash)]
	keys := make([]int64, 0, len(vms))
	for k := range vms {
		keys = append(keys, k)
	}
	// remove from cache
	for _, k := range keys {
		v.cache.Remove(k)
	}
}

// Len returns total number of cached VM instances.
func (v *VmCache) Len() int {
	v.lock.RLock()
	defer v.lock.RUnlock()
	return v.cache.Len()
}
