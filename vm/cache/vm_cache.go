package vmcache

import (
	"fmt"
	"github.com/go-interpreter/wagon/exec"
	"github.com/hashicorp/golang-lru"
	"sync"
)

const DefaultLruSize = 100

type VmCache struct {
	cache   *lru.Cache
	counter int64
	byName  map[string]map[int64]bool
	byIndex map[int64]string
	lock    sync.RWMutex
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

func buildKey(first, second string) string {
	const sep = "|"
	return fmt.Sprintf("%s%s%s", first, sep, second)
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

func (v *VmCache) Put(owner, contract string, vm *exec.VM) {
	v.lock.Lock()
	defer v.lock.Unlock()
	v.counter++
	k, s := v.counter, buildKey(owner, contract)
	vms := v.byName[s]
	if vms == nil {
		vms = make(map[int64]bool)
		v.byName[s] = vms
	}
	vms[k] = true
	v.byIndex[k] = s
	v.cache.Add(k, vm)
}

func (v *VmCache) Fetch(owner, contract string) (vm *exec.VM) {
	v.lock.Lock()
	defer v.lock.Unlock()
	if vms := v.byName[buildKey(owner, contract)]; len(vms) > 0 {
		var key int64
		for k := range vms {
			key = k
			break
		}
		if val, ok := v.cache.Peek(key); ok {
			vm = val.(*exec.VM)
			v.cache.Remove(key)
		}
	}
	return
}

func (v *VmCache) Contains(owner, contract string) int {
	v.lock.RLock()
	defer v.lock.RUnlock()
	return len(v.byName[buildKey(owner, contract)])
}

func (v *VmCache) Remove(owner, contract string) {
	v.lock.Lock()
	defer v.lock.Unlock()
	vms := v.byName[buildKey(owner, contract)]
	keys := make([]int64, 0, len(vms))
	for k := range vms {
		keys = append(keys, k)
	}
	for _, k := range keys {
		v.cache.Remove(k)
	}
}

func (v *VmCache) Len(owner, contract string) int {
	v.lock.RLock()
	defer v.lock.RUnlock()
	return v.cache.Len()
}
