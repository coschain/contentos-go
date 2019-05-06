package iservices

import "github.com/coschain/contentos-go/iservices/itype"

var DailyStatisticServiceName = "dailystatistic"

type IDailyStats interface {
	DailyStatsOn(date string, dapp string) *itype.Row
	DailyStatsSince(days int, dapp string) []*itype.Row
}