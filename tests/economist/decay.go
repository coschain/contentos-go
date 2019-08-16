package economist

import (
	"github.com/coschain/contentos-go/common/constants"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

type DecayTester struct {
	acc0,acc1,acc2 *DandelionAccount
}

func (tester *DecayTester) Test(t *testing.T, d *Dandelion) {

	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	a := assert.New(t)
	a.NoError(tester.acc2.SendTrxAndProduceBlock(TransferToVest(tester.acc2.Name, tester.acc2.Name, constants.MinBpRegisterVest, "")))
	a.NoError(tester.acc2.SendTrxAndProduceBlock(BpRegister(tester.acc2.Name, "", "", tester.acc2.GetPubKey(), mintProps)))

	const VEST = 1000

	a.NoError(tester.acc0.SendTrx(TransferToVest(tester.acc0.Name, tester.acc0.Name, VEST, "")))
	a.NoError(tester.acc1.SendTrx(TransferToVest(tester.acc1.Name, tester.acc1.Name, VEST, "")))

	t.Run("init", d.Test(tester.normal))
	t.Run("decay post", d.Test(tester.decayPost))
	t.Run("decay reply", d.Test(tester.decayReply))
	t.Run("decay vote", d.Test(tester.decayVote))
}

func (tester *DecayTester) normal(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const BLOCK = 100
	a.NoError(d.ProduceBlocks(BLOCK))
	postWeightedVps := d.GlobalProps().PostWeightedVps
	replyWeightedVps := d.GlobalProps().ReplyWeightedVps
	voteWeightedVps := d.GlobalProps().VoteWeightedVps

	a.NoError(d.ProduceBlocks(1))

	// only vote weighted
	a.Equal(d.GlobalProps().PostWeightedVps, bigDecay(StringToBigInt(postWeightedVps)).String())
	a.Equal(d.GlobalProps().ReplyWeightedVps, bigDecay(StringToBigInt(replyWeightedVps)).String())
	a.Equal(d.GlobalProps().VoteWeightedVps, bigDecay(StringToBigInt(voteWeightedVps)).String())
}

func (tester *DecayTester) decayPost(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const POST = 1
	const BLOCKS = 100

	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	a.NoError(d.ProduceBlocks(BLOCKS))
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, POST)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - BLOCKS - 1))

	oldReplyWeightedVps := d.GlobalProps().ReplyWeightedVps
	oldVoteWeightedVps := d.GlobalProps().VoteWeightedVps

	postWeight := StringToBigInt(d.Post(1).GetWeightedVp())
	bigTotalPostWeight, _ := new(big.Int).SetString(d.GlobalProps().GetPostWeightedVps(), 10)
	decayedPostWeight := bigDecay(bigTotalPostWeight)
	exceptNextBlockPostWeightedVps := decayedPostWeight.Add(decayedPostWeight, postWeight)

	a.NoError(d.ProduceBlocks(1))

	a.Equal(d.GlobalProps().PostWeightedVps, exceptNextBlockPostWeightedVps.String())
	a.Equal(d.GlobalProps().ReplyWeightedVps, bigDecay(StringToBigInt(oldReplyWeightedVps)).String())
	a.Equal(d.GlobalProps().VoteWeightedVps,  bigDecay(StringToBigInt(oldVoteWeightedVps)).String())
}

func (tester *DecayTester) decayReply(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const POST = 2
	const BLOCKS = 100
	const REPLY = 3

	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Reply(REPLY, POST,  tester.acc0.Name, "content",  nil)))
	a.NoError(d.ProduceBlocks(BLOCKS))
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, POST)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - BLOCKS - 1))

	oldPostWeightedVps := d.GlobalProps().PostWeightedVps
	oldVoteWeightedVps := d.GlobalProps().VoteWeightedVps

	replyWeight := StringToBigInt(d.Post(REPLY).GetWeightedVp())
	bigTotalReplyWeight, _ := new(big.Int).SetString(d.GlobalProps().GetReplyWeightedVps(), 10)
	decayedReplyWeight := bigDecay(bigTotalReplyWeight)
	exceptNextBlockReplyWeightedVps := decayedReplyWeight.Add(decayedReplyWeight, replyWeight)

	a.NoError(d.ProduceBlocks(1))

	a.Equal(d.GlobalProps().PostWeightedVps, bigDecay(StringToBigInt(oldPostWeightedVps)).String())
	a.Equal(d.GlobalProps().ReplyWeightedVps, exceptNextBlockReplyWeightedVps.String())
	a.Equal(d.GlobalProps().VoteWeightedVps,  bigDecay(StringToBigInt(oldVoteWeightedVps)).String())
}

func (tester *DecayTester) decayVote(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const POST = 4
	const BLOCKS = 100

	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	a.NoError(d.ProduceBlocks(BLOCKS))
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, POST)))
	a.NoError(d.ProduceBlocks(constants.VoteCashOutDelayBlock - 1))

	oldPostWeightedVps := d.GlobalProps().PostWeightedVps
	oldReplyWeightedVps := d.GlobalProps().ReplyWeightedVps

	postWeightedVp := StringToBigInt(d.Post(POST).GetWeightedVp())
	voteWeightedVp := StringToBigInt(d.Vote(tester.acc1.Name, POST).GetWeightedVp())
	weightedVp := new(big.Int).Mul(postWeightedVp, voteWeightedVp)
	decayedVoteWeight := bigDecay(StringToBigInt(d.GlobalProps().GetVoteWeightedVps()))
	totalVoteWeightedVp := decayedVoteWeight.Add(decayedVoteWeight, weightedVp)

	a.NoError(d.ProduceBlocks(1))

	a.Equal(d.GlobalProps().PostWeightedVps, bigDecay(StringToBigInt(oldPostWeightedVps)).String())
	a.Equal(d.GlobalProps().ReplyWeightedVps, bigDecay(StringToBigInt(oldReplyWeightedVps)).String())
	a.Equal(d.GlobalProps().VoteWeightedVps, totalVoteWeightedVp.String())
}

