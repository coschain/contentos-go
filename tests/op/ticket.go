package op

import (
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/prototype"
	"github.com/stretchr/testify/assert"
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
	ops = append(ops,TransferToVest(constants.COSInitMiner, "actor0", constants.MinBpRegisterVest))
	ops = append(ops,TransferToVest(constants.COSInitMiner, "actor1", constants.MinBpRegisterVest))
	ops = append(ops,TransferToVest(constants.COSInitMiner, "actor2", constants.MinBpRegisterVest))

	ops = append(ops,Stake(constants.COSInitMiner,"actor0",1))
	ops = append(ops,Stake(constants.COSInitMiner,"actor1",1))
	ops = append(ops,Stake(constants.COSInitMiner,"actor2",1))
	a.NoError(checkError(d.Account(constants.COSInitMiner).TrxReceipt(ops...)))
	resetProperties(&defaultProps)

	t.Run("normal", d.Test(tester.normal))

}

func (tester *TicketTester) normal(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	balance0 := tester.acc0.GetBalance().Value

	op := AcquireTicket(tester.acc0.Name, 1)
	a.NoError(tester.acc0.SendTrxAndProduceBlock(op)) // ##block 1

	props := d.GlobalProps()
	a.Equal(tester.acc0.GetBalance().Value+props.PerTicketPrice.Value, balance0)
	a.Equal(props.TicketsIncome.Value, props.PerTicketPrice.Value)
	a.Equal(props.ChargedTicketsNum, uint64(1))
	ticketKey := &prototype.GiftTicketKeyType{
		Type: 1,
		From: "contentos",
		To: tester.acc0.Name,
		CreateBlock: props.HeadBlockNumber,
	}
	ticketWrap := table.NewSoGiftTicketWrap(tester.acc0.D.Database(), ticketKey)
	a.Empty(!ticketWrap.CheckExist())

	op = Post(1, tester.acc0.Name, "title", "content", []string{"1"}, make(map[string]int))
	a.NoError(tester.acc0.SendTrx(op))
	op = VoteByTicket(tester.acc0.Name, 1, 1)
	a.NoError(tester.acc0.SendTrxAndProduceBlock(op)) // ##block 2

	props = d.GlobalProps()
}