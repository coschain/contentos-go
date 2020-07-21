package iservices

import "github.com/coschain/contentos-go/iservices/itype"

var DailyStatisticServiceName = "dailystatservice"

type IDailyStats interface {
	DailyStatsSince(days int, dapp string) []*itype.Row
	MonthlyStatsSince(months int, dapp string) []*itype.MonthlyInfo
}