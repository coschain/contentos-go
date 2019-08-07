package app

import (
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
		dgpo.TotalCos.Sub(value)
		dgpo.TotalVest.Add(value.ToVest())
	})
}

func (dgp *DynamicGlobalPropsRW) TransferFromVest(value *prototype.Vest) {
	dgp.ModifyProps(func(dgpo *prototype.DynamicProperties) {
		dgpo.TotalCos.Add(value.ToCoin())
		dgpo.TotalVest.Sub(value)
	})
}

func (dgp *DynamicGlobalPropsRW) TransferToStakeVest(value *prototype.Coin) {
	dgp.ModifyProps(func(dgpo *prototype.DynamicProperties) {
		dgpo.StakeVest.Add(value.ToVest())
	})
}

func (dgp *DynamicGlobalPropsRW) TransferFromStakeVest(value *prototype.Vest) {
	dgp.ModifyProps(func(dgpo *prototype.DynamicProperties) {
		//vest := dgpo.GetStakeVest()
		//mustNoError(vest.Sub(value), "UnStakeVest overflow")
		dgpo.StakeVest.Sub(value)
	})
}


func (dgp *DynamicGlobalPropsRW) ModifyProps(modifier func(oldProps *prototype.DynamicProperties)) {
	dgpWrap := table.NewSoGlobalWrap(dgp.db, &SingleId)
	props := dgpWrap.GetProps()
	modifier(props)
	mustSuccess(dgpWrap.MdProps(props), "")
}

func (dgp *DynamicGlobalPropsRW) UpdateTicketIncomeAndNum(income *prototype.Vest, count uint64) {
	dgp.ModifyProps(func(props *prototype.DynamicProperties) {
		props.TicketsIncome = income
		props.ChargedTicketsNum = count
	})
}
