package blocklog

import (
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/prototype"
)

type ChangeDataMaker func(id, before, after interface{}) *GenericChange

type InterestedChange struct {
	Table *table.TableInfo
	Field string
	Maker ChangeDataMaker
}

var sInterestedChanges = []InterestedChange{
	{
		Table: table.AccountTable,
		Field: "Balance",
		Maker: func(id, before, after interface{}) *GenericChange {
			return &GenericChange{
				Id: 	id.(*prototype.AccountName).GetValue(),
				Before: before.(*prototype.Coin).GetValue(),
				After: 	after.(*prototype.Coin).GetValue(),
			}
		},
	},
	{
		Table: table.AccountTable,
		Field: "Vest",
		Maker: func(id, before, after interface{}) *GenericChange {
			return &GenericChange{
				Id: 	id.(*prototype.AccountName).GetValue(),
				Before: before.(*prototype.Vest).GetValue(),
				After: 	after.(*prototype.Vest).GetValue(),
			}
		},
	},
	{
		Table: table.VoteTable,
		Field: "WeightedVp",
		Maker: func(id, before, after interface{}) *GenericChange {
			cid := id.(*prototype.VoterId)
			return &GenericChange{
				Id: 	cid.PostId,
				Before: before,
				After: 	after,
			}
		},
	},
	{
		Table: table.AccountTable,
		Field: "StakeVestFromMe",
		Maker: func(id, before, after interface{}) *GenericChange {
			return &GenericChange{
				Id: 	id.(*prototype.AccountName).GetValue(),
				Before: before.(*prototype.Vest).GetValue(),
				After: 	after.(*prototype.Vest).GetValue(),
			}
		},
	},
	{
		Table: table.PostTable,
		Field: "WeightedVp",
		Maker: func(id, before, after interface{}) *GenericChange {
			return &GenericChange{
				Id: 	id.(*uint64),
				Before: before.(string),
				After: 	after.(string),
			}
		},
	},
	{
		Table: table.PostTable,
		Field: "CashoutBlockNum",
		Maker: func(id, before, after interface{}) *GenericChange {
			return &GenericChange{
				Id: 	id.(*uint64),
				Before: before.(uint64),
				After: 	after.(uint64),
			}
		},
	},
	{
		Table: table.ContractTable,
		Field: "Balance",
		Maker: func(id, before, after interface{}) *GenericChange {
			cid := id.(*prototype.ContractId)
			return &GenericChange{
				Id: 	cid.Owner.Value + "@" + cid.Cname,
				Before: before.(*prototype.Coin).GetValue(),
				After: 	after.(*prototype.Coin).GetValue(),
			}
		},
	},
	{
		Table: table.StakeRecordTable,
		Field: "LastStakeTime",
		Maker: func(id, before, after interface{}) *GenericChange {
			cid := id.(*prototype.StakeRecord)
			return &GenericChange{
				Id: 	cid.From.Value + "&" + cid.To.Value,
				Before: before.(*prototype.TimePointSec).UtcSeconds,
				After: 	after.(*prototype.TimePointSec).UtcSeconds,
			}
		},
	},
	//{
	//	Table: table.BlockProducerTable,
	//	Field: "BpVest",
	//	Maker: func(id, before, after interface{}) *GenericChange {
	//		return &GenericChange{
	//			Id: 	id.(*prototype.AccountName).Value,
	//			Before: before,
	//			After: 	after,
	//		}
	//	},
	//},
}
