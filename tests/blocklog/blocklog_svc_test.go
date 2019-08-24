//+build !tests

package blocklog

import (
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/iservices"
	"testing"
)

func TestBlockLogService(t *testing.T) {
	t.Run("block_log_svc",
		NewDandelionTestWithPlugins(true, []string{iservices.BlockLogServiceName},
			new(BlockLogTester).Test, sBlockLogTestActors))
}
