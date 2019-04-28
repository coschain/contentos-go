package iservices

var DailyStatisticServiceName = "dailystatistic"

type Row map[string]int

type IDailyStats interface {
	DAUStatsOn(date string) Row
	DAUStatsSince(days int) map[string]Row
	DNUStatsOn(date string) Row
	DNUStatsSince(days int) map[string]Row
}