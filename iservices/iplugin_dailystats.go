package iservices

import "github.com/coschain/contentos-go/iservices/itype"

var DailyStatisticServiceName = "dailystatistic"

type IDailyStats interface {
	DAUStatsOn(date string, dapp string) *itype.Row
	DAUStatsSince(days int, dapp string) []*itype.Row
	DNUStatsOn(date string, dapp string) *itype.Row
	DNUStatsSince(days int, dapp string) []*itype.Row
}