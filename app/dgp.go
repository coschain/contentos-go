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

func (dgp *DynamicGlobalPropsRW) AddWeightedVP(value uint64) {
	dgp.ModifyProps(func(dgpo *prototype.DynamicProperties) {
		dgpo.WeightedVps += value
	})
}

func (dgp *DynamicGlobalPropsRW) ModifyProps(modifier func(oldProps *prototype.DynamicProperties)) {
	dgpWrap := table.NewSoGlobalWrap(dgp.db, &SingleId)
	props := dgpWrap.GetProps()
	modifier(props)
	mustSuccess(dgpWrap.MdProps(props), "")
}
