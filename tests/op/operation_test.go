package op

import (
	"github.com/coschain/contentos-go/dandelion/utils"
	"testing"
)

func TestOperations(t *testing.T) {
	t.Run("transfer", utils.NewDandelionTest(new(TransferTester).Test))
}
