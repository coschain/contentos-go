package iservices

import "github.com/coschain/contentos-go/iservices/itype"

var DailyStatisticServiceName = "dailystatistic"

type IDailyStats interface {
	DAUStatsOn(date string) *itype.Row
	DAUStatsSince(days int) []*itype.Row
	DNUStatsOn(date string) *itype.Row
	DNUStatsSince(days int) []*itype.Row
}