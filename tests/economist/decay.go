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

	registerBlockProducer(tester.acc2, t)

	const VEST = 1000
	SelfTransferToVesting([]*DandelionAccount{tester.acc0, tester.acc1}, VEST, t)

	t.Run("init", d.Test(tester.normal))
	t.Run("decay post", d.Test(tester.decayPost))
	t.Run("decay reply", d.Test(tester.decayReply))
	t.Run("decay vote", d.Test(tester.decayVote))
}

func (tester *DecayTester) normal(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	postWeightedVps := d.GlobalProps().WeightedVpsPost
	replyWeightedVps := d.GlobalProps().WeightedVpsReply
	voteWeightedVps := d.GlobalProps().WeightedVpsVote
	dappWeightedVps := d.GlobalProps().WeightedVpsDapp

	a.NoError(d.ProduceBlocks(1))

	// only vote weighted
	a.Equal(d.GlobalProps().WeightedVpsPost, bigDecay(StringToBigInt(postWeightedVps)).String())
	a.Equal(d.GlobalProps().WeightedVpsReply, bigDecay(StringToBigInt(replyWeightedVps)).String())
	a.Equal(d.GlobalProps().WeightedVpsVote, bigDecay(StringToBigInt(voteWeightedVps)).String())
	a.Equal(d.GlobalProps().WeightedVpsDapp, bigDecay(StringToBigInt(dappWeightedVps)).String())
}

func (tester *DecayTester) decayPost(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const POST = 1

	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, POST)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - 2))

	oldReplyWeightedVps := d.GlobalProps().WeightedVpsReply
	oldVoteWeightedVps := d.GlobalProps().WeightedVpsVote
	oldDappWeightedVps := d.GlobalProps().WeightedVpsDapp

	postWeight := StringToBigInt(d.Post(1).GetWeightedVp())
	a.NotEqual(postWeight.Int64(), int64(0))

	bigTotalPostWeight, _ := new(big.Int).SetString(d.GlobalProps().GetWeightedVpsPost(), 10)
	decayedPostWeight := bigDecay(bigTotalPostWeight)
	exceptNextBlockPostWeightedVps := decayedPostWeight.Add(decayedPostWeight, postWeight)

	a.NoError(d.ProduceBlocks(1))

	a.Equal(d.GlobalProps().WeightedVpsPost, exceptNextBlockPostWeightedVps.String())
	a.Equal(d.GlobalProps().WeightedVpsReply, bigDecay(StringToBigInt(oldReplyWeightedVps)).String())
	a.Equal(d.GlobalProps().WeightedVpsVote,  bigDecay(StringToBigInt(oldVoteWeightedVps)).String())
	a.Equal(d.GlobalProps().WeightedVpsDapp, bigDecay(StringToBigInt(oldDappWeightedVps)).String())
}

func (tester *DecayTester) decayReply(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const POST = 2
	const REPLY = 3

	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Reply(REPLY, POST,  tester.acc0.Name, "content",  nil)))
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, POST)))
	a.NoError(d.ProduceBlocks(constants.PostCashOutDelayBlock - 2))

	oldPostWeightedVps := d.GlobalProps().WeightedVpsPost
	oldVoteWeightedVps := d.GlobalProps().WeightedVpsVote
	oldDappWeightedVps := d.GlobalProps().WeightedVpsDapp

	replyWeight := StringToBigInt(d.Post(REPLY).GetWeightedVp())
	bigTotalReplyWeight, _ := new(big.Int).SetString(d.GlobalProps().GetWeightedVpsReply(), 10)
	decayedReplyWeight := bigDecay(bigTotalReplyWeight)
	exceptNextBlockReplyWeightedVps := decayedReplyWeight.Add(decayedReplyWeight, replyWeight)

	a.NoError(d.ProduceBlocks(1))

	a.Equal(d.GlobalProps().WeightedVpsPost, bigDecay(StringToBigInt(oldPostWeightedVps)).String())
	a.Equal(d.GlobalProps().WeightedVpsReply, exceptNextBlockReplyWeightedVps.String())
	a.Equal(d.GlobalProps().WeightedVpsVote,  bigDecay(StringToBigInt(oldVoteWeightedVps)).String())
	a.Equal(d.GlobalProps().WeightedVpsDapp, bigDecay(StringToBigInt(oldDappWeightedVps)).String())
}

func (tester *DecayTester) decayVote(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const POST = 4

	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, nil)))
	a.NoError(tester.acc1.SendTrxAndProduceBlock(Vote(tester.acc1.Name, POST)))
	a.NoError(d.ProduceBlocks(constants.VoteCashOutDelayBlock - 2))

	oldPostWeightedVps := d.GlobalProps().WeightedVpsPost
	oldReplyWeightedVps := d.GlobalProps().WeightedVpsReply
	oldDappWeightedVps := d.GlobalProps().WeightedVpsDapp

	weightedVp := StringToBigInt(d.Vote(tester.acc1.Name, POST).GetWeightedVp())
	decayedVoteWeight := bigDecay(StringToBigInt(d.GlobalProps().GetWeightedVpsVote()))
	totalVoteWeightedVp := decayedVoteWeight.Add(decayedVoteWeight, weightedVp)

	a.NoError(d.ProduceBlocks(1))

	a.Equal(d.GlobalProps().WeightedVpsPost, bigDecay(StringToBigInt(oldPostWeightedVps)).String())
	a.Equal(d.GlobalProps().WeightedVpsReply, bigDecay(StringToBigInt(oldReplyWeightedVps)).String())
	a.Equal(d.GlobalProps().WeightedVpsVote, totalVoteWeightedVp.String())
	a.Equal(d.GlobalProps().WeightedVpsDapp, bigDecay(StringToBigInt(oldDappWeightedVps)).String())
}

func (tester *DecayTester) decayDapp(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const POST = 4

	beneficiary := []map[string]int{{tester.acc0.Name: 10000}}
	a.NoError(tester.acc0.SendTrxAndProduceBlock(Post(POST, tester.acc0.Name, "title", "content", []string{"1"}, beneficiary)))
	a.NoError(tester.acc1.SendTrx(Vote(tester.acc1.Name, POST)))
	a.NoError(d.ProduceBlocks(constants.VoteCashOutDelayBlock - 2))

	oldReplyWeightedVps := d.GlobalProps().WeightedVpsReply
	oldVoteWeightedVps := d.GlobalProps().WeightedVpsVote

	postWeightedVp := StringToBigInt(d.Post(POST).GetWeightedVp())
	decayGlobalPostWeightedVp := bigDecay(StringToBigInt(d.GlobalProps().GetWeightedVpsPost()))
	nextGlobalPostWeightedVp := new(big.Int).Add(decayGlobalPostWeightedVp, postWeightedVp)

	postDappWeightedVp := StringToBigInt(d.Post(POST).GetWeightedVp())
	decayedGlobalVoteWeightedVp := bigDecay(StringToBigInt(d.GlobalProps().GetWeightedVpsDapp()))
	nextGlobalDappWeightedVp := new(big.Int).Add(decayedGlobalVoteWeightedVp, postDappWeightedVp)

	a.NoError(d.ProduceBlocks(1))

	a.Equal(d.GlobalProps().WeightedVpsPost, nextGlobalPostWeightedVp)
	a.Equal(d.GlobalProps().WeightedVpsReply, bigDecay(StringToBigInt(oldReplyWeightedVps)).String())
	a.Equal(d.GlobalProps().WeightedVpsVote, bigDecay(StringToBigInt(oldVoteWeightedVps)).String())
	a.Equal(d.GlobalProps().WeightedVpsDapp, nextGlobalDappWeightedVp.String())
}