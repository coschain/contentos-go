package op

import (
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/prototype"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"

	. "github.com/coschain/contentos-go/dandelion"
)

type TicketTester struct {
	acc0, acc1, acc2 *DandelionAccount
}

func (tester *TicketTester) Test(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")
	var ops []*prototype.Operation
	ops = append(ops,TransferToVest(constants.COSInitMiner, "actor0", constants.MinBpRegisterVest, ""))
	ops = append(ops,TransferToVest(constants.COSInitMiner, "actor1", constants.MinBpRegisterVest, ""))
	ops = append(ops,TransferToVest(constants.COSInitMiner, "actor2", constants.MinBpRegisterVest, ""))

	ops = append(ops,Stake(constants.COSInitMiner,"actor0",1))
	ops = append(ops,Stake(constants.COSInitMiner,"actor1",1))
	ops = append(ops,Stake(constants.COSInitMiner,"actor2",1))
	a.NoError(checkError(d.Account(constants.COSInitMiner).TrxReceipt(ops...)))

	t.Run("normal", d.Test(tester.normal))
	t.Run("invalidAcquire", d.Test(tester.invalidAcquireOp))
	t.Run("invalidVote", d.Test(tester.invalidVoteOp))
}

func (tester *TicketTester) normal(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	balance0 := tester.acc0.GetBalance().Value

	// buy ticket and check
	op := AcquireTicket(tester.acc0.Name, 1)
	a.NoError(tester.acc0.SendTrxAndProduceBlock(op)) // ##block 1
	props := d.GlobalProps()
	a.Equal(tester.acc0.GetBalance().Value+props.PerTicketPrice.Value, balance0)
	a.Equal(props.TicketsIncome.Value, props.PerTicketPrice.Value)
	a.Equal(props.ChargedTicketsNum, uint64(1))
	ticketKey := &prototype.GiftTicketKeyType{
		Type: 1,
		From: constants.COSSysAccount,
		To: tester.acc0.Name,
		CreateBlock: props.HeadBlockNumber,
	}
	ticketWrap := table.NewSoGiftTicketWrap(tester.acc0.D.Database(), ticketKey)
	a.Empty(!ticketWrap.CheckExist())
	a.Equal(tester.acc0.GetChargedTicket(), uint32(1))

	// vote ticket and check
	op = Post(1, tester.acc1.Name, "title", "content", []string{"1"}, nil)
	a.NoError(tester.acc1.SendTrxAndProduceBlock(op))  // ##block 2
	props = d.GlobalProps()
	op = VoteByTicket(tester.acc0.Name, 1, 1)
	valOfTicket := &prototype.Vest{Value: props.TicketsIncome.Value/props.ChargedTicketsNum}
	valOfTicket.Mul(1)
	a.NoError(tester.acc0.SendTrxAndProduceBlock(op)) // ##block 3
	props = d.GlobalProps()
	a.Equal(tester.acc0.GetChargedTicket(), uint32(0))
	ticketKey = &prototype.GiftTicketKeyType{
		Type: 1,
		From: tester.acc0.Name,
		To: strconv.FormatUint(1, 10),
		CreateBlock: props.HeadBlockNumber,
	}
	ticketWrap = table.NewSoGiftTicketWrap(tester.acc0.D.Database(), ticketKey)
	a.Empty(!ticketWrap.CheckExist())
	// TODO: check current bp properties
}

func (tester *TicketTester) invalidAcquireOp(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	// count = 0
	op := AcquireTicket(tester.acc0.Name, 0)
	a.Error(checkError(tester.acc0.TrxReceipt(op)))
	d.ProduceBlocks(1)

	// count exceeds max limit
	op = AcquireTicket(tester.acc0.Name, constants.MaxTicketsPerTurn+1)
	a.Error(checkError(tester.acc0.TrxReceipt(op)))
	d.ProduceBlocks(1)

	balance0 := tester.acc0.GetBalance().Value
	op = TransferToVest(tester.acc0.Name, tester.acc0.Name, balance0, "")
	a.NoError(checkError(tester.acc0.TrxReceipt(op)))
	d.ProduceBlocks(1)

	// not enough fund
	op = AcquireTicket(tester.acc0.Name, 1)
	a.Error(checkError(tester.acc0.TrxReceipt(op)))
	d.ProduceBlocks(1)
}

func (tester *TicketTester) invalidVoteOp(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	const POSTID uint64 = 20

	a.NoError( d.Account(constants.COSInitMiner).SendTrxAndProduceBlock( Transfer(constants.COSInitMiner, tester.acc0.Name, 2 * constants.PerTicketPrice * constants.COSTokenDecimals, "")) )
	a.True( tester.acc0.GetBalance().Value > constants.PerTicketPrice * constants.COSTokenDecimals)
	op := AcquireTicket(tester.acc0.Name, 1)
	a.NoError(tester.acc0.SendTrxAndProduceBlock(op)) // ##block 1
	op = Post(POSTID, tester.acc1.Name, "title", "content", []string{"1"}, nil)
	a.NoError(tester.acc1.SendTrxAndProduceBlock(op))  // ##block 2

	// count = 0
	op = VoteByTicket(tester.acc0.Name, POSTID, 0)
	a.Error(checkError(tester.acc0.TrxReceipt(op)))

	// count exceeds limit
	op = VoteByTicket(tester.acc0.Name, POSTID, 2)
	a.Error(checkError(tester.acc0.TrxReceipt(op)))

	// vote for a non-existed post
	op = VoteByTicket(tester.acc0.Name, POSTID + 1, 1)
	a.Error(checkError(tester.acc0.TrxReceipt(op)))
}

func (tester *TicketTester) freeTicket(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	tester.acc1.D.ProduceBlocks(constants.MinEpochDuration)
	props := d.GlobalProps()
	freeTicketWrap := table.NewSoGiftTicketWrap(tester.acc1.D.Database(), &prototype.GiftTicketKeyType{
		Type: 0,
		From: constants.COSSysAccount,
		To: tester.acc1.Name,
		CreateBlock: props.GetCurrentEpochStartBlock(),
	})
	a.Empty(!freeTicketWrap.CheckExist())

	op := Post(2, tester.acc0.Name, "title", "content", []string{"1"}, nil)
	a.NoError(tester.acc0.SendTrxAndProduceBlock(op))
	op = VoteByTicket(tester.acc0.Name, 2, 1)
	a.Error(checkError(tester.acc0.TrxReceipt(op)))
}