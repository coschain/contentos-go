package op

import (
	"github.com/coschain/contentos-go/common/constants"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

type VoteTester struct {
	acc0, acc1, acc2 *DandelionAccount
}

func (tester *VoteTester) TestNormal(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	t.Run("normal", d.Test(tester.normal))
}

func (tester *VoteTester) TestRevote(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	t.Run("revote", d.Test(tester.revote))
}

func (tester *VoteTester) normal(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.NoError(tester.acc0.SendTrx(Post(1, tester.acc0.Name, "title", "content", []string{"1"}, make(map[string]int))))
	a.NoError(d.ProduceBlocks(1))
	// the init status, vp bar equals 0, so the vote power is 10 * 0 == 0
	a.NoError(tester.acc0.SendTrx(Vote(tester.acc0.Name, 1)))
	a.NoError(d.ProduceBlocks(1))
	a.Equal("0", d.Post(1).GetWeightedVp())
	// after 1000 blocks, vp bar should be recovered to 1000 * 1000 / constants.VoteRegenerateTime
	// and used current vp should be (currentVp + constants.VoteLimitDuringRegenerate - 1) / constants.VoteLimitDuringRegenerate
	a.NoError(d.ProduceBlocks(1000))
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, 1)))
	a.NoError(d.ProduceBlocks(1))
	currentVp := 1000 * 1000 / constants.VoteRegenerateTime
	usedVp := (currentVp + constants.VoteLimitDuringRegenerate - 1) / constants.VoteLimitDuringRegenerate
	a.Equal(strconv.FormatUint(uint64(usedVp) * tester.acc1.GetVestingShares().Value, 10), d.Post(1).GetWeightedVp())
}

func (tester *VoteTester) revote(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.NoError(tester.acc0.SendTrx(Post(1, tester.acc0.Name, "title", "content", []string{"1"}, make(map[string]int))))
	a.NoError(d.ProduceBlocks(1))
	// the init status, vp bar equals 0, so the vote power is 10 * 0 == 0
	a.NoError(tester.acc0.SendTrx(Vote(tester.acc0.Name, 1)))
	a.NoError(d.ProduceBlocks(1))
	a.Equal("0", d.Post(1).GetWeightedVp())
	a.NoError(tester.acc0.SendTrx(Vote(tester.acc0.Name, 1)))
	a.NoError(d.ProduceBlocks(1))
}