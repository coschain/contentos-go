package utils

type IConsumer interface {
	Consume(name string, num uint64, now uint64) bool
}

type IGetter interface {
	Get(name string) uint64
	GetCapacity(name string) uint64
}

// stake resource interface
type IStakeManager interface {
	IConsumer
	IGetter
	LockUpCos(name string, cos uint64) bool
	ReleaseCos(name string, cos uint64)
}

// free resource interface
type IFreeManager interface {
	// recover free net value, return true if success
	IConsumer
	IGetter
}

// StakeManager impl all IStakeManager's api
type StakeManager struct {

}

func NewStakeManager() *StakeManager {
	return &StakeManager{}
}

// FreeManager impl all IFreeManager's api
type FreeManager struct {

}

func NewFreeManager() *FreeManager {
	return &FreeManager{}
}

/* below is pseudo code */

const CHAIN_STAMINA = 100000000

const PRECISION = 10000

// recover minimum time gap
const MIN_RECOVER_DURATION = 45

// resource recover in every 24H
const RECOVER_WINDOW = 60 * 60 * 24

// free resource every 24H
const FREE_STAMINA = 10000

// StakeManager implemention
func (s *StakeManager) LockUpCos(name string, cos uint64) bool {
	/*  1.get account from db
			2.transfer cos to cpu vesting
			3.update db
		*/
		if _,ok := db[name]; !ok {
			return  false
		}
		a,_ := db[name]
		if a.cos < cos{
			return false
		}
		a.cos -= cos
		a.vest += cos
		global.totalVest += cos
		return true
}

func (s *StakeManager) ReleaseCos(name string, cos uint64) bool {
	/*  1.get account from db
			2.transfer cpu vesting to cos
			3.update db
		*/
	if _,ok := db[name]; !ok {
		return false
	}
	a,_ := db[name]
	if a.vest < cos{
		return false
	}
	a.cos += cos
	a.vest -= cos
	global.totalVest -= cos
	return true
}

func (s *StakeManager) Get(name string) uint64{
	/*  1.get account resource from db
			2.return resource.cpu
		*/
		if _,ok := db[name];!ok {
			return 0
		}
		return db[name].stamina
}

func (c *StakeManager) GetCapacity(name string) uint64 {
	return calculateUserMaxStamina(name)
}

func (c *StakeManager) Consume(name string, num uint64, now uint64) bool {
	/*  1.get account resource from db
			2.resource.cpu -= num
			3.resource.cpu = min(resource.cpu,0)
			3.if (resource.cpu - num >= 0) return true else return false
		*/
	if _,ok := db[name];!ok {
		return false
	}
	a,_ := db[name]
	newStamina := calculateNewStamina(a.stamina,0,a.staminaUseTime,now)
	maxStamina := calculateUserMaxStamina(a.name)
	if maxStamina - newStamina < num {
		return false
	}
	newStamina = calculateNewStamina(newStamina,num,now,now)

	db[name].stamina = newStamina
	db[name].staminaUseTime = now
	return true
}

// FreeManager implemention
func (f *FreeManager) Consume(name string,num uint64, now uint64) bool {
	/*  1.get account resource from db
		2.resource.cpuFree -= num
		3.resource.cpuFree = min(resource.cpuFree,0)
		3.if (resource.cpuFree - num >= 0) return true else return false
	*/
	if _,ok := db[name];!ok {
		return false
	}
	a,_ := db[name]
	newFreeStamina := calculateNewStamina(a.staminaFree,0,a.staminaFreeUseTime,now)
	if uint64(FREE_STAMINA) - newFreeStamina < num {
		return false
	}
	newFreeStamina = calculateNewStamina(newFreeStamina,num,now,now)

	db[name].staminaFree = newFreeStamina
	db[name].staminaFreeUseTime = now
	return true
}

func (f *FreeManager) Get(name string) uint64 {
	/*  1.get account resource from db
		2.return resource.cpuFree
	*/
	if _,ok := db[name];!ok {
		return 0
	}
	return db[name].staminaFree
}

func (c *FreeManager) GetCapacity(name string) uint64 {
	return FREE_STAMINA
}