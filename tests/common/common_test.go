package common

import (
	"github.com/coschain/contentos-go/dandelion"
	"testing"
)

func TestCommons(t *testing.T) {
	t.Run("trx", dandelion.NewDandelionTest(new(TrxTester).Test, 3))
}
