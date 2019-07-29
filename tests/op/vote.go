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
	t.Run("vote to ghost post", d.Test(tester.voteToGhostPost))
}

func (tester *VoteTester) TestFullPower(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	t.Run("fullpower", d.Test(tester.fullPower))
	t.Run("voteaftercashout", d.Test(tester.voteAfterPostCashout))
}

func (tester *VoteTester) normal(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(1, tester.acc0.Name, "title", "content", []string{"1"}, make(map[string]int))))
	// the init status, vp bar equals 0, so the vote power is 10 * 0 == 0
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Vote(tester.acc0.Name, 1)))
	a.Equal("0", d.Post(1).GetWeightedVp())
	// after 1000 blocks, vp bar should be recovered to 1000 * 1000 / constants.VoteRegenerateTime
	// and used current vp should be (currentVp + constants.VoteLimitDuringRegenerate - 1) / constants.VoteLimitDuringRegenerate
	BLOCKS := 100
	a.NoError(d.ProduceBlocks(BLOCKS))
	a.NoError(tester.acc1.SendTrxAndProduceBlock(Vote(tester.acc1.Name, 1)))
	currentVp := BLOCKS * 1000 / constants.VoteRegenerateTime
	usedVp := (currentVp + constants.VoteLimitDuringRegenerate - 1) / constants.VoteLimitDuringRegenerate
	a.Equal(strconv.FormatUint(uint64(usedVp) * tester.acc1.GetVestingShares().Value, 10), d.Post(1).GetWeightedVp())
}

func (tester *VoteTester) revote(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(1, tester.acc0.Name, "title", "content", []string{"1"}, make(map[string]int))))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Vote(tester.acc0.Name, 1)))
	a.Equal("0", d.Post(1).GetWeightedVp())
	receipt, err := tester.acc0.SendTrxEx(Vote(tester.acc0.Name, 1))
	a.NoError(err)
	a.NotEqual(receipt.Status, SUCCESS)
}

func (tester *VoteTester) voteToGhostPost(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	receipt, err := tester.acc0.SendTrxEx(Vote(tester.acc0.Name, 2))
	a.NoError(err)
	a.NotEqual(receipt.Status, SUCCESS)
}


func (tester *VoteTester) fullPower(t *testing.T, d *Dandelion)  {
	a := assert.New(t)
	// waiting vote power recover
	BLOCKS := 10000
	a.NoError(d.ProduceBlocks(BLOCKS))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(1, tester.acc0.Name, "title", "content", []string{"1"}, make(map[string]int))))
	a.NoError(tester.acc1.SendTrxAndProduceBlock(Vote(tester.acc1.Name, 1)))
	currentVp := 1000
	usedVp := (currentVp + constants.VoteLimitDuringRegenerate - 1) / constants.VoteLimitDuringRegenerate
	a.Equal(strconv.FormatUint(uint64(usedVp) * tester.acc1.GetVestingShares().Value, 10), d.Post(1).GetWeightedVp())
}

func (tester *VoteTester) voteAfterPostCashout(t *testing.T, d *Dandelion)  {
	a := assert.New(t)
	// waiting vote power recover
	oldVp := d.Post(1).GetWeightedVp()
	BLOCKS := int(constants.PostCashOutDelayBlock)
	a.NoError(d.ProduceBlocks(BLOCKS))
	a.NoError(tester.acc1.SendTrxAndProduceBlock(Vote(tester.acc1.Name, 1)))
	a.Equal(oldVp, d.Post(1).GetWeightedVp())
}