package op

import (
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
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

	t.Run("follow", d.Test(tester.follow))
	t.Run("unfollow", d.Test(tester.unfollow))
	t.Run("follow self", d.Test(tester.followSelf))
	t.Run("unfollow no related", d.Test(tester.unfollowUnrelated))
	t.Run("follow to no exist", d.Test(tester.followNoExist))
	t.Run("follow use other private key", d.Test(tester.followUseOther))
}

// the follow evaluator doesn't apply anything
func (tester *FollowTester) follow(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Follow(tester.acc0.Name, tester.acc1.Name, false)))
}

func (tester *FollowTester) unfollow(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Follow(tester.acc0.Name, tester.acc1.Name, true)))
}

func (tester *FollowTester) unfollowUnrelated(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Follow(tester.acc0.Name, tester.acc2.Name, true)))
}

func (tester *FollowTester) followSelf(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	receipt, err := tester.acc0.SendTrxEx(Follow(tester.acc0.Name, tester.acc0.Name, false))
	a.NoError(err)
	a.NotEqual(receipt.Status, prototype.StatusSuccess)
}

func (tester *FollowTester) followNoExist(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	receipt, err := tester.acc0.SendTrxEx(Follow(tester.acc0.Name, "actor4", false))
	a.NoError(err)
	a.NotEqual(receipt.Status, prototype.StatusSuccess)
}

func (tester *FollowTester) followUseOther(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.Error(tester.acc0.SendTrxAndProduceBlock(Follow(tester.acc1.Name, tester.acc0.Name, false)))
}