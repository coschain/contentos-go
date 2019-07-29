package contracts

import (
	"testing"
)

func TestContracts(t *testing.T) {
	t.Run("natives", NewDandelionContractTest(new(NativeTester).Test, 2, "actor1.native_tester"))
}
