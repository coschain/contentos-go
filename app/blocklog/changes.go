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
}
