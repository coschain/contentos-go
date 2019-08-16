package op

import (
	"github.com/coschain/contentos-go/common/constants"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

type VoteTester struct {
	acc0, acc1, acc2 *DandelionAccount
}

func ISqrt(n uint64) uint64 {
	if n == 0 {
		return 0
	}
	var r1, r uint64 = n, n + 1
	for r1 < r {
		r, r1 = r1, (r1+n/r1)>>1
	}
	return r
}

func (tester *VoteTester) TestNormal(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	t.Run("normal", d.Test(tester.normal))
	t.Run("normal", d.Test(tester.voteSelf))
}

func (tester *VoteTester) TestRevote(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	t.Run("revote", d.Test(tester.revote))
	t.Run("vote to ghost post", d.Test(tester.voteToGhostPost))
}

func (tester *VoteTester) TestZeroPower(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	t.Run("fullpower", d.Test(tester.zeroPower))
}

func (tester *VoteTester) TestVoteAfterCashout(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	t.Run("voteaftercashout", d.Test(tester.voteAfterPostCashout))
}

func (tester *VoteTester) normal(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	const POST1 = 1
	const POST2 = 2
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST1, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST2, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	a.NoError(tester.acc1.SendTrxAndProduceBlock(Vote(tester.acc1.Name, POST1)))
	usedVp := uint32(constants.FullVP / constants.VPMarks)
	a.Equal(strconv.FormatUint(uint64(usedVp) * ISqrt(tester.acc1.GetVest().Value), 10), d.Post(POST1).GetWeightedVp())
	a.NoError(tester.acc1.SendTrxAndProduceBlock(Vote(tester.acc1.Name, POST2)))
	a.Equal(strconv.FormatUint(uint64(usedVp) * ISqrt(tester.acc1.GetVest().Value), 10), d.Post(POST2).GetWeightedVp())
}


func (tester *VoteTester) voteSelf(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	const POST1 = 11
	const REPLY1 = 12
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST1, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Reply(REPLY1, POST1, tester.acc0.Name,  "content", nil)))

	a.Equal( d.TrxReceiptByAccount( tester.acc0.Name, Vote(tester.acc0.Name, POST1) ).Status , prototype.StatusDeductStamina)
	a.Equal( d.TrxReceiptByAccount( tester.acc0.Name, Vote(tester.acc0.Name, REPLY1) ).Status , prototype.StatusDeductStamina)
}

func (tester *VoteTester) revote(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	const POST = 1
	a.NoError(tester.acc1.SendTrxAndProduceBlock(Post(1, tester.acc1.Name, "title", "content", []string{"1"}, nil)))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Vote(tester.acc0.Name, 1)))
	usedVp := uint32(constants.FullVP / constants.VPMarks)
	a.Equal(strconv.FormatUint(uint64(usedVp) * ISqrt(tester.acc0.GetVest().Value), 10), d.Post(POST).GetWeightedVp())
	receipt, err := tester.acc0.SendTrxEx(Vote(tester.acc0.Name, 1))
	a.NoError(err)
	a.NotEqual(receipt.Status, prototype.StatusSuccess)
}

func (tester *VoteTester) voteToGhostPost(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	receipt, err := tester.acc0.SendTrxEx(Vote(tester.acc0.Name, 2))
	a.NoError(err)
	a.NotEqual(receipt.Status, prototype.StatusSuccess)
}


func (tester *VoteTester) zeroPower(t *testing.T, d *Dandelion)  {
	a := assert.New(t)
	// waiting vote power recover
	i := 1
	for i < 40 {
		a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(uint64(i), tester.acc0.Name, "title", "content", []string{"1"}, nil)))
		a.NoError(tester.acc1.SendTrxAndProduceBlock(Vote(tester.acc1.Name, uint64(i))))
		i ++
	}
	a.Equal("0", d.Post(35).GetWeightedVp())
	a.Equal(uint32(1000 - 33 * 30), d.Account(tester.acc1.Name).GetVotePower())
}

func (tester *VoteTester) voteAfterPostCashout(t *testing.T, d *Dandelion)  {
	a := assert.New(t)
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(uint64(1), tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	a.NoError(tester.acc1.SendTrxAndProduceBlock(Vote(tester.acc1.Name, uint64(1))))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock))
	// waiting vote power recover
	oldVp := d.Post(1).GetWeightedVp()
	accountVP := d.Account(tester.acc1.Name).GetVotePower()
	BLOCKS := int(constants.PostCashOutDelayBlock)
	a.NoError(d.ProduceBlocks(BLOCKS))
	a.NoError(tester.acc1.SendTrxAndProduceBlock(Vote(tester.acc1.Name, 1)))
	a.Equal(oldVp, d.Post(1).GetWeightedVp())
	a.Equal(accountVP, d.Account(tester.acc1.Name).GetVotePower())
}