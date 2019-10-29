package iservices

var IpRestrictServiceName = "iprestrict"

type IIpRestrict interface {
	AddToWhiteList(ip string)

	AddToBlackList(ip string)

	IsValidIp(ip string) bool
	UpdateMonitor(ip string)
	HitWhiteList(ip string) bool

	HitBlackList(ip string) bool

	HitMonitorList(ip string) bool
}
