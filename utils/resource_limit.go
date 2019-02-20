package utils

// stake resource limiter
type IStakeManager interface {
	LockUpCos(name string, cos uint64)
	ReleaseCos(name string, cos uint64)
	Get(name string) uint64
	Recover(name string)
}
// consumer resource
type IConsumer interface {
	Consume(name string, num uint64)
}

// free resource limiter
type IFreeManager interface {
	// recover free net value, return true if success
	RecoverFree(name string)
	ConsumeFree(name string, num uint64)
	GetFree(name string)
}

// StakeManager impl all IStakeManager's api
type StakeManager struct {

}

// FreeManager impl all IFreeManager's api
type FreeManager struct {

}

// cpu use StakeManager and impl cpu Consume api
type CpuManager struct {
	StakeManager
	FreeManager
	IConsumer
}

// net use StakeManager and impl net Consume api
type NetManager struct {
	StakeManager
	FreeManager
	IConsumer
}

/* below is pseudo code */

// recover minimum time gap
const MIN_RECOVER_DURATION = 60*60

// resource recover in every 24H
const RECOVER_WINDOW = 60 * 60 * 24

// free cpu resource value
const FREE_CPU_VALUE = 10000

// resource present each account's resource
type Resource struct {
	Name string
	Stamina uint64
	StaminaFree uint64
	StaminaUseTime uint32
	StaminaFreeUseTime uint32
}

// StakeManager implemention
func (s *StakeManager) LockUpCos(name string, cos uint64) {
	/*  1.get account from db
			2.transfer cos to cpu vesting
			3.update db
		*/
}

func (s *StakeManager) ReleaseCos(name string, cos uint64) {
	/*  1.get account from db
			2.transfer cpu vesting to cos
			3.update db
		*/
}

func (s *StakeManager) Get(name string) {
	/*  1.get account resource from db
			2.return resource.cpu
		*/
}

func (s *StakeManager) Recover(name string) {
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

// FreeManager implemention
func (f *FreeManager) RecoverFree(name string) {
	/*  1.get account resource from db
		2.calculate recover value according to formula:
			newfreecpu = (now - cpuFreeLastUseTime) > 0 ? FREE_CPU_VALUE : resource.cpuFree
		3.resource.cpu = newfreecpu
	*/
}

func (f *FreeManager) ConsumeFree(name string) {
	/*  1.get account resource from db
		2.resource.cpuFree -= num
		3.resource.cpuFree = min(resource.cpuFree,0)
		3.if (resource.cpuFree - num >= 0) return true else return false
	*/
}

func (f *FreeManager) GetFree(name string) {
	/*  1.get account resource from db
		2.return resource.cpuFree
	*/
}

func (c *CpuManager) Consume(name string, num uint64) {
	/*  1.get account resource from db
			2.resource.cpu -= num
			3.resource.cpu = min(resource.cpu,0)
			3.if (resource.cpu - num >= 0) return true else return false
		*/
}

func (n *NetManager) Consume(name string, num uint64) {
	/*  1.get account resource from db
			2.resource.cpu -= num
			3.resource.cpu = min(resource.cpu,0)
			3.if (resource.cpu - num >= 0) return true else return false
		*/
}