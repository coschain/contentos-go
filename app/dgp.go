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
		cos := dgpo.GetTotalCos()
		vest := dgpo.GetTotalVest()
		addVest := value.ToVest()

		mustNoError(cos.Sub(value), "TotalCos overflow")
		dgpo.TotalCos = cos

		mustNoError(vest.Add(addVest), "TotalVest overflow")
		dgpo.TotalVest = vest
	})
}

func (dgp *DynamicGlobalPropsRW) TransferFromVest(value *prototype.Vest) {
	dgp.ModifyProps(func(dgpo *prototype.DynamicProperties) {
		cos := dgpo.GetTotalCos()
		vest := dgpo.GetTotalVest()
		addCos := value.ToCoin()

		mustNoError(cos.Add(addCos), "TotalCos overflow")
		dgpo.TotalCos = cos

		mustNoError(vest.Sub(value), "TotalVest overflow")
		dgpo.TotalVest = vest
	})
}

func (dgp *DynamicGlobalPropsRW) TransferToStakeVest(value *prototype.Coin) {
	dgp.ModifyProps(func(dgpo *prototype.DynamicProperties) {
		vest := dgpo.GetStakeVest()
		addVest := value.ToVest()

		mustNoError(vest.Add(addVest), "StakeVest overflow")
		dgpo.StakeVest = vest
	})
}

func (dgp *DynamicGlobalPropsRW) TransferFromStakeVest(value *prototype.Vest) {
	dgp.ModifyProps(func(dgpo *prototype.DynamicProperties) {
		vest := dgpo.GetStakeVest()

		mustNoError(vest.Sub(value), "UnStakeVest overflow")
		dgpo.StakeVest = vest
	})
}


func (dgp *DynamicGlobalPropsRW) ModifyProps(modifier func(oldProps *prototype.DynamicProperties)) {
	dgpWrap := table.NewSoGlobalWrap(dgp.db, &SingleId)
	props := dgpWrap.GetProps()
	modifier(props)
	dgpWrap.SetProps(props)
}

func (dgp *DynamicGlobalPropsRW) UpdateTicketIncomeAndNum(income *prototype.Vest, count uint64) {
	dgp.ModifyProps(func(props *prototype.DynamicProperties) {
		props.TicketsIncome = income
		props.ChargedTicketsNum = count
	})
}
