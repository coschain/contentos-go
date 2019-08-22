package economist

import (
	"github.com/coschain/contentos-go/dandelion"
	"testing"
)

func TestEconomist(t *testing.T) {
	t.Run("mint", dandelion.NewDandelionTest(new(MintTester).Test, 3))
	t.Run("post", dandelion.NewDandelionTest(new(PostTester).Test1, 3))
	t.Run("post", dandelion.NewDandelionTest(new(PostTester).Test2, 3))
	t.Run("post", dandelion.NewDandelionTest(new(PostTester).Test3, 3))
	t.Run("post", dandelion.NewDandelionTest(new(PostTester).Test4, 3))
	t.Run("dapp", dandelion.NewDandelionTest(new(DappTester).Test, 5))
	t.Run("reply", dandelion.NewDandelionTest(new(ReplyTester).Test1, 3))
	t.Run("reply", dandelion.NewDandelionTest(new(ReplyTester).Test2, 3))
	t.Run("reply", dandelion.NewDandelionTest(new(ReplyTester).Test3, 3))
	t.Run("reply", dandelion.NewDandelionTest(new(ReplyTester).Test4, 3))
	t.Run("vote", dandelion.NewDandelionTest(new(VoteTester).Test, 5))
	t.Run("decay", dandelion.NewDandelionTest(new(DecayTester).Test, 3))
	t.Run("util", dandelion.NewDandelionTest(new(UtilTester).Test, 3))
}


