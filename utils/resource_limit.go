package utils

// net resource limiter
type NetManager interface {
	// lock up cos to net vesting for net resource
	LockCosToNet(name string, cos uint64) bool
	// release up cos from net vesting
	ReleaseCosFromNet(name string, cos uint64) bool
	// get net resource value
	GetNet(name string) uint64
	// recover net value, return true if recover value > 0
	RecoverNet(name string) bool
	// consume net value, return true if account left value >= num
	ConsumeNet(name string, num uint64) bool
}

// cpu resource limiter
type CpuManager interface {
	LockCosToCpu(name string, cos uint64) bool
	ReleaseCosFromCpu(name string, cos uint64) bool
	GetCpu(name string) uint64
	RecoverCpu(name string) bool
	ConsumeCpu(name string, num uint64) bool
}

// free resource limiter
type FreeManager interface {
	// recover free net value, return true if success
	RecoverFreeNet(name string) bool
	RecoverFreeCpu(name string) bool
	ConsumeFreeNet(name string, num uint64) bool
	ConsumeFreeCpu(name string, num uint64) bool
	GetFreeNet(name string) bool
	GetFreeCpu(name string) bool
}

type ResourceManager interface {
	NetManager
	CpuManager
	FreeManager
}

/* below is pseudo code */

// recover minimum time gap
const MIN_RECOVER_DURATION = 60*60

// resource recover in every 24H
const RECOVER_WINDOW = 60 * 60 * 24

// free cpu resource value
const FREE_CPU_VALUE = 10000

// resource present each account's resource
type resource struct {
	name string
	cpu,net uint64
	cpuFree,netFree uint64
	cpuLastUseTime,netLastUseTime uint32
	cpuFreeLastUseTime,netFreeLastUseTime uint32
}

// implemention of ResourceManager
type ResourceManagerImpl struct {

}

func (rm *ResourceManagerImpl) LockCosToCpu(name string, cos uint64) {
	/*  1.get account from db
		2.transfer cos to cpu vesting
		3.update db
	*/
}

func (rm *ResourceManagerImpl) ReleaseCosFromCpu(name string, cos uint64) {
	/*  1.get account from db
		2.transfer cpu vesting to cos
		3.update db
	*/
}

func (rm *ResourceManagerImpl) GetCpu(name string) {
	/*  1.get account resource from db
		2.return resource.cpu
	*/
}

func (rm *ResourceManagerImpl) RecoverCpu(name string) {
	/*  1.get all system vesting from db
		2.get account from db
		3.get account cpu vesting
		4.accountCpu = (accountVesting/allVesting) * virtualMaxCpuValue
		5.calculate recover value according to formula:
			newcpu = cpu + ((now - cpuLastUseTime) / RECOVER_WINDOW) * accountCpu
			newcpu = max(newcpu,accountCpu)
		6.get account resource from db, resource.cpu += newcpu
	*/
}

func (rm *ResourceManagerImpl) ConsumeCpu(name string, num uint64) {
	/*  1.get account resource from db
		2.resource.cpu -= num
		3.resource.cpu = min(resource.cpu,0)
		3.if (resource.cpu - num >= 0) return true else return false
	*/
}

func (rm *ResourceManagerImpl) RecoverFreeCpu(name string) {
	/*  1.get account resource from db
		2.calculate recover value according to formula:
			newfreecpu = (now - cpuFreeLastUseTime) > 0 ? FREE_CPU_VALUE : resource.cpuFree
		3.resource.cpu = newfreecpu
	*/
}

func (rm *ResourceManagerImpl) ConsumeFreeCpu(name string) {
	/*  1.get account resource from db
		2.resource.cpuFree -= num
		3.resource.cpuFree = min(resource.cpuFree,0)
		3.if (resource.cpuFree - num >= 0) return true else return false
	*/
}

func (rm *ResourceManagerImpl) GetFreeCpu(name string) {
	/*  1.get account resource from db
		2.return resource.cpuFree
	*/
}