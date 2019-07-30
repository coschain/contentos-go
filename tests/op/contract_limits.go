package op

import (
	"fmt"
	"github.com/coschain/contentos-go/common/constants"
	. "github.com/coschain/contentos-go/dandelion"
	"testing"
)

type ContractLimitsTester struct {}

func (tester *ContractLimitsTester) Test(t *testing.T, d *Dandelion) {
	t.Run("big_memory", d.Test(tester.big_memory))
	t.Run("call_depth", d.Test(tester.call_depth))
	t.Run("infinite_loop", d.Test(tester.infinite_loop))
}

func (tester *ContractLimitsTester) big_memory(t *testing.T, d *Dandelion) {
	ApplyNoError(t, d, fmt.Sprintf("actor0: actor0.limits.alloc_mem %d", 10 * 1024))
	ApplyNoError(t, d, fmt.Sprintf("actor0: actor0.limits.alloc_mem %d", 1024 * 1024))
	ApplyNoError(t, d, fmt.Sprintf("actor0: actor0.limits.alloc_mem %d", 25 * 1024 * 1024))
	ApplyError(t, d, fmt.Sprintf("actor0: actor0.limits.alloc_mem %d", 36 * 1024 * 1024))
	ApplyError(t, d, fmt.Sprintf("actor0: actor0.limits.alloc_mem %d", 100 * 1024 * 1024))
}

func (tester *ContractLimitsTester) call_depth(t *testing.T, d *Dandelion) {
	ApplyNoError(t, d, fmt.Sprintf("actor0: actor0.limits.call_depth %d", 5))
	ApplyNoError(t, d, fmt.Sprintf("actor0: actor0.limits.call_depth %d", 100))
	ApplyNoError(t, d, fmt.Sprintf("actor0: actor0.limits.call_depth %d", 200))
	ApplyError(t, d, fmt.Sprintf("actor0: actor0.limits.call_depth %d", 260))
	ApplyError(t, d, fmt.Sprintf("actor0: actor0.limits.call_depth %d", 1024))
}

func (tester *ContractLimitsTester) infinite_loop(t *testing.T, d *Dandelion) {
	ApplyError(t, d, fmt.Sprintf("%s: actor0.limits.infinite_loop", "actor0"))
	ApplyError(t, d, fmt.Sprintf("%s: actor0.limits.infinite_loop", constants.COSInitMiner))
}
