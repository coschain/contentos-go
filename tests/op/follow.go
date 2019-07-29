package op

import (
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/stretchr/testify/assert"
	"testing"
)

type FollowTester struct {
	acc0, acc1, acc2 *DandelionAccount
}

func (tester *FollowTester) Test(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	t.Run("normal", d.Test(tester.follow))
	t.Run("normal", d.Test(tester.unfollow))
	t.Run("normal", d.Test(tester.followSelf))
	t.Run("normal", d.Test(tester.followNoExist))
	t.Run("normal", d.Test(tester.followerUseOther))
}

func (tester *FollowTester) follow(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Follow("actor0", "actor1", false)))
}

func (tester *FollowTester) unfollow(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Follow("actor0", "actor1", true)))
}

func (tester *FollowTester) followSelf(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	receipt, err := tester.acc0.SendTrxEx(Follow("actor0", "actor0", false))
	a.NoError(err)
	a.NotEqual(receipt.Status, 200)
}

func (tester *FollowTester) followNoExist(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	receipt, err := tester.acc0.SendTrxEx(Follow("actor0", "actor4", false))
	a.NoError(err)
	a.NotEqual(receipt.Status, 200)
}

func (tester *FollowTester) followerUseOther(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	receipt, err := tester.acc0.SendTrxEx(Follow("actor1", "actor0", false))
	a.NoError(err)
	a.NotEqual(receipt.Status, 200)
}