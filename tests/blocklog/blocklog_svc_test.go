//+build !tests

package blocklog

import (
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/iservices"
	"testing"
	"time"
)

func TestBlockLogService(t *testing.T) {
	t.Run("block_log_svc",
		NewDandelionTestWithPlugins(true, []string{iservices.BlockLogServiceName, iservices.BlockLogProcessServiceName},
			new(BlockLogServiceTester).Test, sBlockLogTestActors))
}

type BlockLogServiceTester struct {}

func (tester *BlockLogServiceTester) Test(t *testing.T, d *Dandelion) {
	new(BlockLogTester).Test(t, d)
	// sleep a while so that block log process service has some time to work
	time.Sleep(5 * time.Second)
}
