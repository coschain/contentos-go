package op

import (
	"fmt"
	"github.com/coschain/contentos-go/common/constants"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
	"time"
)

const TicketBpBonusActors = 30

type TicketBpBonusTester struct {
	a *assert.Assertions
	d *Dandelion
	actors []*DandelionAccount
	expected map[string]uint64
	hasBonus bool
}

func (tester *TicketBpBonusTester) Test(t *testing.T, d *Dandelion) {
	tester.d, tester.a = d, assert.New(t)
	for i := 0; i < TicketBpBonusActors; i++ {
		tester.actors = append(tester.actors, d.Account(fmt.Sprintf("actor%d", i)))
	}

	//
	// Ticket bonus distribution is done in pre-shuffle stage.
	// We subscribe before/afterPreShuffle event which fires before and after ticket bonus distribution.
	// We predict distribution result in beforePreShuffle() and examine it in afterPreShuffle()
	//
	d.SubscribePreShuffle(true, tester.beforePreShuffle)
	d.SubscribePreShuffle(false, tester.afterPreShuffle)
	defer func() {
		d.UnsubscribePreShuffle(true, tester.beforePreShuffle)
		d.UnsubscribePreShuffle(false, tester.afterPreShuffle)
	}()

	// the main entry
	tester.testMain()
}

func (tester *TicketBpBonusTester) testMain() {
	const epoch = constants.MaxBlockProducerCount * constants.BlockProdRepetition

	a := tester.a
	rand.Seed(time.Now().UnixNano())

	// all actors become block producers
	tester.newBlockProducers()

	// repeat for a few times: post -> buy tickets -> vote by tickets
	for i := 0; i < 100; i++ {
		tester.postAndVote()
		// random intervals
		a.NoError(tester.d.ProduceBlocks(rand.Intn(epoch)))
	}
	// make sure we have enough blocks so that all ticket bonus could be distributed
	a.NoError(tester.d.ProduceBlocks(epoch))
}

func (tester *TicketBpBonusTester) beforePreShuffle() {
	a := tester.a

	//
	// simple equal distribution among block producers
	//
	tester.expected = make(map[string]uint64)
	bonus := tester.d.GlobalProps().GetTicketsBpBonus()
	a.Equal(tester.hasBonus, bonus.Value > 0)
	names, _, _ := tester.d.TrxPool().GetShuffledBpList()
	share, remain := prototype.NewVest(bonus.Value / uint64(len(names))), prototype.NewVest(bonus.Value % uint64(len(names)))
	for i, name := range names {
		vest := tester.d.Account(name).GetVest()
		vest.Add(share)
		if i == 0 {
			vest.Add(remain)
		}
		tester.expected[name] = vest.Value
	}
	tester.hasBonus = false
}

func (tester *TicketBpBonusTester) afterPreShuffle() {
	a := tester.a

	// ticket bonus should always be 0 after distribution
	a.EqualValues(0, tester.d.GlobalProps().GetTicketsBpBonus().Value)

	// check distribution results
	for name, vest := range tester.expected {
		a.EqualValues(vest, tester.d.Account(name).GetVest().Value)
	}
}

func (tester *TicketBpBonusTester) newBlockProducers() {
	a := tester.a
	var ops []*prototype.Operation
	bpInitminer := tester.d.BlockProducer(constants.COSInitMiner)
	chainProp := &prototype.ChainProperties{
		AccountCreationFee:   bpInitminer.GetAccountCreateFee(),
		StaminaFree:          bpInitminer.GetProposedStaminaFree(),
		TpsExpected:          bpInitminer.GetTpsExpected(),
		TopNAcquireFreeToken: bpInitminer.GetTopNAcquireFreeToken(),
		EpochDuration:        bpInitminer.GetEpochDuration(),
		PerTicketPrice:       bpInitminer.GetPerTicketPrice(),
		PerTicketWeight:      bpInitminer.GetPerTicketWeight(),
	}
	ops = append(ops, Stake(constants.COSInitMiner, constants.COSInitMiner, 10000 * constants.COSTokenDecimals))
	for _, actor := range tester.actors {
		ops = append(ops, Stake(constants.COSInitMiner, actor.Name, 10000 * constants.COSTokenDecimals))
		ops = append(ops, TransferToVest(constants.COSInitMiner, actor.Name, constants.MinBpRegisterVest, ""))
	}
	r := tester.d.Account(constants.COSInitMiner).TrxReceipt(ops...)
	a.True(r != nil && r.Status == prototype.StatusSuccess)

	for _, actor := range tester.actors {
		r := actor.TrxReceipt(
			BpRegister(actor.Name, "http://foo", "blabla", actor.GetPubKey(), chainProp),
			BpVote(actor.Name, actor.Name, false))
		a.True(r != nil && r.Status == prototype.StatusSuccess)
	}
}

func (tester *TicketBpBonusTester) hasFreeTicket(name string) bool {
	return tester.d.GiftTicket(&prototype.GiftTicketKeyType{
		Type: 0,
		From: constants.COSSysAccount,
		To: name,
		CreateBlock: tester.d.GlobalProps().GetCurrentEpochStartBlock(),
	}).CheckExist()
}

func (tester *TicketBpBonusTester) postAndVote() {
	a := tester.a

	// initminer posts an article
	postId := tester.d.GlobalProps().HeadBlockNumber
	r := tester.d.Account(constants.COSInitMiner).TrxReceipt(Post(
		postId,
		constants.COSInitMiner,
		fmt.Sprintf("title %d", postId),
		fmt.Sprintf("content %d", postId),
		[]string{"tag1", "tag2"},
		nil))
	a.True(r != nil && r.Status == prototype.StatusSuccess)

	// a random actor buys random number of tickets and vote for the article using these tickets
	actor := tester.actors[rand.Intn(TicketBpBonusActors)]
	ticketCount := uint64(rand.Intn(3) + 1)

	// if the actor votes using 1 ticket and he has a free ticket, the free ticket will be used.
	// in which case, block producers bonus should not change.
	if !tester.hasBonus {
		if ticketCount > 1 || !tester.hasFreeTicket(actor.Name) {
			tester.hasBonus = true
		}
	}

	r = actor.TrxReceipt(
		AcquireTicket(actor.Name, ticketCount),
		VoteByTicket(actor.Name, postId, ticketCount))
	a.True(r != nil && r.Status == prototype.StatusSuccess)
}
