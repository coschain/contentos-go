package blocklog

import (
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/prototype"
	"reflect"
)

const (
	AccountBalance = "account_balance"
	AccountVest = "account_vest"
)

type ChangeDataMaker func(key, before, after interface{}) interface{}

type InterestedChange struct {
	what string
	record reflect.Type
	primary string
	field string
	maker ChangeDataMaker
}

var sInterestedChanges = []InterestedChange{
	{ AccountBalance, table.AccountRecordType, "Name", "Balance", accountBalanceChange },
	{ AccountVest, table.AccountRecordType, "Name", "Vest", accountVestChange },
}

type Uint64ValueChange struct {
	Id 		string			`json:"id"`
	Before	uint64			`json:"before"`
	After	uint64			`json:"after"`
}

func accountBalanceChange(key, before, after interface{}) interface{} {
	return &Uint64ValueChange{
		Id: 	key.(*prototype.AccountName).GetValue(),
		Before: before.(*prototype.Coin).GetValue(),
		After: 	after.(*prototype.Coin).GetValue(),
	}
}

func accountVestChange(key, before, after interface{}) interface{} {
	return &Uint64ValueChange{
		Id: 	key.(*prototype.AccountName).GetValue(),
		Before: before.(*prototype.Vest).GetValue(),
		After: 	after.(*prototype.Vest).GetValue(),
	}
}
