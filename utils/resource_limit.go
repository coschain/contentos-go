package utils

import (
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
)

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
	db      iservices.IDatabaseService
}

func NewStakeManager(db iservices.IDatabaseService) *StakeManager {
	return &StakeManager{db:db}
}

// FreeManager impl all IFreeManager's api
type FreeManager struct {
	db      iservices.IDatabaseService
}

func NewFreeManager(db iservices.IDatabaseService) *FreeManager {
	return &FreeManager{db:db}
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
	accountWrap := table.NewSoAccountWrap(s.db, &prototype.AccountName{Value:name})
	if !accountWrap.CheckExist() {
		return 0
	}
	return accountWrap.GetStamina()
}

func (s *StakeManager) GetCapacity(name string) uint64 {
	return s.calculateUserMaxStamina(name)
}

var SINGLE int32 = 1
func (s *StakeManager) calculateUserMaxStamina(name string) uint64 {
	accountWrap := table.NewSoAccountWrap(s.db, &prototype.AccountName{Value:name})
	if !accountWrap.CheckExist() {
		return 0
	}
	dgpWrap := table.NewSoGlobalWrap(s.db,&SINGLE)

	userMax := float64(accountWrap.GetVestingShares().Value * CHAIN_STAMINA)/float64(dgpWrap.GetProps().TotalVestingShares.Value)
	return uint64(userMax)
}

func (s *StakeManager) Consume(name string, num uint64, now uint64) bool {
	accountWrap := table.NewSoAccountWrap(s.db, &prototype.AccountName{Value:name})
	if !accountWrap.CheckExist() {
		return false
	}

	newStamina := calculateNewStamina(accountWrap.GetStamina(),0,accountWrap.GetStaminaUseBlock(),now)
	maxStamina := s.calculateUserMaxStamina(name)
	if maxStamina - newStamina < num {
		return false
	}
	newStamina = calculateNewStamina(newStamina,num,now,now)

	accountWrap.MdStamina(newStamina)
	accountWrap.MdStaminaUseBlock(now)
	return true
}

func (s *StakeManager) ConsumeLeft(name string, now uint64) bool {
	accountWrap := table.NewSoAccountWrap(s.db, &prototype.AccountName{Value:name})
	if !accountWrap.CheckExist() {
		return false
	}

	maxStamina := s.calculateUserMaxStamina(name)

	accountWrap.MdStamina(maxStamina)
	accountWrap.MdStaminaUseBlock(now)
	return true
}

// FreeManager implemention
func (f *FreeManager) Consume(name string,num uint64, now uint64) bool {
	accountWrap := table.NewSoAccountWrap(f.db, &prototype.AccountName{Value:name})
	if !accountWrap.CheckExist() {
		return false
	}
	newFreeStamina := calculateNewStamina(accountWrap.GetStaminaFree(),0,accountWrap.GetStaminaFreeUseBlock(),now)
	if uint64(FREE_STAMINA) - newFreeStamina < num {
		return false
	}
	newFreeStamina = calculateNewStamina(newFreeStamina,num,now,now)

	accountWrap.MdStaminaFree(newFreeStamina)
	accountWrap.MdStaminaFreeUseBlock(now)
	return true
}

func (f *FreeManager) Get(name string) uint64 {
	accountWrap := table.NewSoAccountWrap(f.db, &prototype.AccountName{Value:name})
	if !accountWrap.CheckExist() {
		return 0
	}
	return accountWrap.GetStaminaFree()
}

func (f *FreeManager) GetCapacity(name string) uint64 {
	return FREE_STAMINA
}

func calculateNewStamina(oldStamina uint64, useStamina uint64, lastTime uint64, now uint64) uint64 {
	blocks := uint64(RECOVER_WINDOW/3)
	if now > lastTime { // assert ?
		if now < lastTime + blocks {
			delta := now - lastTime
			decay := float64(blocks - delta) / float64(blocks)
			newStamina := float64(oldStamina) * decay
			oldStamina = uint64(newStamina)
		} else {
			oldStamina = 0
		}
	}
	oldStamina += useStamina
	return oldStamina
}