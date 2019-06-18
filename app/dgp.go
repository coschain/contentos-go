package app

import (
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
)

type DynamicGlobalPropsRW struct {
	db iservices.IDatabaseRW
}

func (dgp *DynamicGlobalPropsRW) GetProps() *prototype.DynamicProperties {
	dgpWrap := table.NewSoGlobalWrap(dgp.db, &SingleId)
	return dgpWrap.GetProps()
}

func (dgp *DynamicGlobalPropsRW) HeadBlockTime() *prototype.TimePointSec {
	return dgp.GetProps().GetTime()
}

func (dgp *DynamicGlobalPropsRW) TransferToVest(value *prototype.Coin) {
	dgp.ModifyProps(func(dgpo *prototype.DynamicProperties) {
		cos := dgpo.GetTotalCos()
		vest := dgpo.GetTotalVestingShares()
		addVest := value.ToVest()

		mustNoError(cos.Sub(value), "TotalCos overflow")
		dgpo.TotalCos = cos

		mustNoError(vest.Add(addVest), "TotalVestingShares overflow")
		dgpo.TotalVestingShares = vest
	})
}

func (dgp *DynamicGlobalPropsRW) TransferFromVest(value *prototype.Vest) {
	dgp.ModifyProps(func(dgpo *prototype.DynamicProperties) {
		cos := dgpo.GetTotalCos()
		vest := dgpo.GetTotalVestingShares()
		addCos := value.ToCoin()

		mustNoError(cos.Add(addCos), "TotalCos overflow")
		dgpo.TotalCos = cos

		mustNoError(vest.Sub(value), "TotalVestingShares overflow")
		dgpo.TotalVestingShares = vest
	})
}

func (dgp *DynamicGlobalPropsRW) TransferToStakeVest(value *prototype.Coin) {
	dgp.ModifyProps(func(dgpo *prototype.DynamicProperties) {
		vest := dgpo.GetStakeVestingShares()
		addVest := value.ToVest()

		mustNoError(vest.Add(addVest), "StakeVestingShares overflow")
		dgpo.StakeVestingShares = vest
	})
}

func (dgp *DynamicGlobalPropsRW) TransferFromStakeVest(value *prototype.Vest) {
	dgp.ModifyProps(func(dgpo *prototype.DynamicProperties) {
		vest := dgpo.GetStakeVestingShares()

		mustNoError(vest.Sub(value), "UnStakeVestingShares overflow")
		dgpo.StakeVestingShares = vest
	})
}


func (dgp *DynamicGlobalPropsRW) ModifyProps(modifier func(oldProps *prototype.DynamicProperties)) {
	dgpWrap := table.NewSoGlobalWrap(dgp.db, &SingleId)
	props := dgpWrap.GetProps()
	modifier(props)
	mustSuccess(dgpWrap.MdProps(props), "")
}

func (dgp *DynamicGlobalPropsRW) TicketFee(fee *prototype.Vest) {
	dgp.ModifyProps(func(dgpo *prototype.DynamicProperties) {
		vest := dgpo.GetTicketFeeToBp()
		mustNoError(vest.Add(fee), "TicketFeeToBp overflow")
		dgpo.TicketFeeToBp = vest
	})
}

func (dgp *DynamicGlobalPropsRW) VoteByTicket(account string, postId uint64, count uint64) {
	currentWitness := dgp.GetProps().CurrentWitness
	bpWrap := table.NewSoAccountWrap(dgp.db, currentWitness)
	if !bpWrap.CheckExist() {
		panic(fmt.Sprintf("cannot find account %s", currentWitness.Value))
	}
	bpVest := bpWrap.GetVestingShares()
	tax := count * dgp.GetProps().GetTicketPrice()
	mustNoError(bpVest.Add(&prototype.Vest{Value:tax}), "add tax to bp failed")
	vest := dgp.GetProps().GetTicketFeeToBp()
	mustNoError(vest.Sub(&prototype.Vest{Value:tax}), "sub tax from ticketfee failed")
	bpWrap.MdVestingShares(bpVest)
	dgp.ModifyProps(func(props *prototype.DynamicProperties) {
		props.TicketFeeToBp = vest
	})
}
