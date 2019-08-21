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

	registerBlockProducer(tester.acc2, t)

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
	postWeightedVps := d.GlobalProps().WeightedVpsPost
	replyWeightedVps := d.GlobalProps().WeightedVpsReply
	voteWeightedVps := d.GlobalProps().WeightedVpsVote

	a.NoError(d.ProduceBlocks(1))

	// only vote weighted
	a.Equal(d.GlobalProps().WeightedVpsPost, bigDecay(StringToBigInt(postWeightedVps)).String())
	a.Equal(d.GlobalProps().WeightedVpsReply, bigDecay(StringToBigInt(replyWeightedVps)).String())
	a.Equal(d.GlobalProps().WeightedVpsVote, bigDecay(StringToBigInt(voteWeightedVps)).String())
}

func (tester *DecayTester) decayPost(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const POST = 1
	const BLOCKS = 100

	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	a.NoError(d.ProduceBlocks(BLOCKS))
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, POST)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - BLOCKS - 1))

	oldReplyWeightedVps := d.GlobalProps().WeightedVpsReply
	oldVoteWeightedVps := d.GlobalProps().WeightedVpsVote

	postWeight := StringToBigInt(d.Post(1).GetWeightedVp())
	a.NotEqual(postWeight.Int64(), int64(0))

	bigTotalPostWeight, _ := new(big.Int).SetString(d.GlobalProps().GetWeightedVpsPost(), 10)
	decayedPostWeight := bigDecay(bigTotalPostWeight)
	exceptNextBlockPostWeightedVps := decayedPostWeight.Add(decayedPostWeight, postWeight)

	a.NoError(d.ProduceBlocks(1))

	a.Equal(d.GlobalProps().WeightedVpsPost, exceptNextBlockPostWeightedVps.String())
	a.Equal(d.GlobalProps().WeightedVpsReply, bigDecay(StringToBigInt(oldReplyWeightedVps)).String())
	a.Equal(d.GlobalProps().WeightedVpsVote,  bigDecay(StringToBigInt(oldVoteWeightedVps)).String())
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

	oldPostWeightedVps := d.GlobalProps().WeightedVpsPost
	oldVoteWeightedVps := d.GlobalProps().WeightedVpsVote

	replyWeight := StringToBigInt(d.Post(REPLY).GetWeightedVp())
	bigTotalReplyWeight, _ := new(big.Int).SetString(d.GlobalProps().GetWeightedVpsReply(), 10)
	decayedReplyWeight := bigDecay(bigTotalReplyWeight)
	exceptNextBlockReplyWeightedVps := decayedReplyWeight.Add(decayedReplyWeight, replyWeight)

	a.NoError(d.ProduceBlocks(1))

	a.Equal(d.GlobalProps().WeightedVpsPost, bigDecay(StringToBigInt(oldPostWeightedVps)).String())
	a.Equal(d.GlobalProps().WeightedVpsReply, exceptNextBlockReplyWeightedVps.String())
	a.Equal(d.GlobalProps().WeightedVpsVote,  bigDecay(StringToBigInt(oldVoteWeightedVps)).String())
}

func (tester *DecayTester) decayVote(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const POST = 4
	const BLOCKS = 100

	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	a.NoError(d.ProduceBlocks(BLOCKS))
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, POST)))
	a.NoError(d.ProduceBlocks(constants.VoteCashOutDelayBlock - 1))

	oldPostWeightedVps := d.GlobalProps().WeightedVpsPost
	oldReplyWeightedVps := d.GlobalProps().WeightedVpsReply

	postWeightedVp := StringToBigInt(d.Post(POST).GetWeightedVp())
	voteWeightedVp := StringToBigInt(d.Vote(tester.acc1.Name, POST).GetWeightedVp())
	weightedVp := new(big.Int).Mul(postWeightedVp, voteWeightedVp)
	decayedVoteWeight := bigDecay(StringToBigInt(d.GlobalProps().GetWeightedVpsVote()))
	totalVoteWeightedVp := decayedVoteWeight.Add(decayedVoteWeight, weightedVp)

	a.NoError(d.ProduceBlocks(1))

	a.Equal(d.GlobalProps().WeightedVpsPost, bigDecay(StringToBigInt(oldPostWeightedVps)).String())
	a.Equal(d.GlobalProps().WeightedVpsReply, bigDecay(StringToBigInt(oldReplyWeightedVps)).String())
	a.Equal(d.GlobalProps().WeightedVpsVote, totalVoteWeightedVp.String())
}

