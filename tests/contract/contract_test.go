package contracts

import (
	"testing"
)

func TestContracts(t *testing.T) {
	t.Run("misc", NewDandelionContractTest(new(MiscTester).Test, 2, "actor0.native_tester", "actor1.native_tester"))
}
