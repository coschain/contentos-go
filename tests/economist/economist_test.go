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
	t.Run("post dapp", dandelion.NewDandelionTest(new(PostDappTester).Test, 5))
	t.Run("reply", dandelion.NewDandelionTest(new(ReplyTester).Test1, 3))
	t.Run("reply", dandelion.NewDandelionTest(new(ReplyTester).Test2, 3))
	t.Run("reply", dandelion.NewDandelionTest(new(ReplyTester).Test3, 3))
	t.Run("reply", dandelion.NewDandelionTest(new(ReplyTester).Test4, 3))
	t.Run("reply dapp", dandelion.NewDandelionTest(new(ReplyDappTester).Test, 5))
	t.Run("vote", dandelion.NewDandelionTest(new(VoteTester).Test, 5))
	t.Run("decay", dandelion.NewDandelionTest(new(DecayTester).Test, 3))
}


