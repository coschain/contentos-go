package vmcache

import (
	"github.com/hashicorp/golang-lru"
	"sync"
	"github.com/go-interpreter/wagon/exec"
)

const DefaultLruSize = 100

type VmCache struct {
	cache *lru.Cache
}

var once sync.Once
var vc *VmCache

func GetVmCache() *VmCache {
	once.Do(func() {
		vc = &VmCache{}
		lru,err := lru.New(DefaultLruSize)
		if err != nil {
			panic(err)
		}
		vc.cache = lru
	})
	return vc
}

func buildKey(first,second string) string {
	return first + second
}

func (v *VmCache) Add(owner,contract string, vm *exec.VM) bool {
	return v.cache.Add(buildKey(owner,contract),vm)
}

func (v *VmCache) Get(owner,contract string) (*exec.VM,bool) {
	value,ok := v.cache.Get(buildKey(owner,contract))
	if !ok {
		return nil,ok
	}
	vm := value.(*exec.VM)
	return vm,ok
}

func (v *VmCache) Contains(owner,contract string) bool {
	return v.cache.Contains(buildKey(owner,contract))
}

func (v *VmCache) Remove(owner,contract string) {
	v.cache.Remove(buildKey(owner,contract))
}

func (v *VmCache) Len(owner,contract string) int {
	return v.cache.Len()
}