package economist

import (
	"github.com/coschain/contentos-go/dandelion"
	"testing"
)

func TestEconomist(t *testing.T) {
	t.Run("mint", dandelion.NewDandelionTest(new(MintTester).Test, 3))
	t.Run("post", dandelion.NewDandelionTest(new(PostTester).Test1, 3))
	t.Run("post", dandelion.NewDandelionTest(new(PostTester).Test2, 3))
}


