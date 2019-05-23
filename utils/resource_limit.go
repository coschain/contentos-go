package utils

import (
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
)

type IConsumer interface {
	Consume(db iservices.IDatabaseRW,name string, num uint64, now uint64) bool
	ConsumeLeft(db iservices.IDatabaseRW,name string, now uint64) bool
}

type IFreeConsumer interface {
	ConsumeFree(db iservices.IDatabaseRW,name string, num uint64, now uint64) bool
	ConsumeFreeLeft(db iservices.IDatabaseRW,name string, now uint64) bool
}

type IGetter interface {
	Get(db iservices.IDatabaseRW,name string) uint64
	GetCapacity(db iservices.IDatabaseRW,name string) uint64
	GetStakeLeft(db iservices.IDatabaseRW,name string, now uint64) uint64
}

type IFreeGetter interface {
	GetFree(db iservices.IDatabaseRW,name string) uint64
	GetCapacityFree() uint64
	GetFreeLeft(db iservices.IDatabaseRW,name string, now uint64) uint64
}

// stake resource interface
type IResourceLimiter interface {
	IConsumer
	IFreeConsumer
	IGetter
	IFreeGetter
}

// ResourceLimiter impl all IResourceLimiter's api
type ResourceLimiter struct {
}

func NewResourceLimiter() IResourceLimiter {
	return IResourceLimiter(&ResourceLimiter{})
}

const PRECISION = 10000

// recover minimum time gap
const MIN_RECOVER_DURATION = 45

// ResourceLimiter implemention
func (s *ResourceLimiter) Get(db iservices.IDatabaseRW,name string) uint64{
	accountWrap := table.NewSoAccountWrap(db, &prototype.AccountName{Value:name})
	if !accountWrap.CheckExist() {
		return 0
	}
	return accountWrap.GetStamina()
}

func (s *ResourceLimiter) GetCapacity(db iservices.IDatabaseRW,name string) uint64 {
	return s.calculateUserMaxStamina(db,name)
}

var SINGLE int32 = 1
func (s *ResourceLimiter) calculateUserMaxStamina(db iservices.IDatabaseRW, name string) uint64 {
	accountWrap := table.NewSoAccountWrap(db, &prototype.AccountName{Value:name})
	if !accountWrap.CheckExist() {
		return 0
	}
	dgpWrap := table.NewSoGlobalWrap(db,&SINGLE)

	stakeVest := accountWrap.GetStakeVesting().Value

	allVest := dgpWrap.GetProps().StakeVestingShares.Value
	if allVest == 0 {
		return 0
	}
	userMax := float64( stakeVest)/float64(allVest) * constants.OneDayStamina
	return uint64(userMax)
}

func (s *ResourceLimiter) Consume(db iservices.IDatabaseRW,name string, num uint64, now uint64) bool {
	accountWrap := table.NewSoAccountWrap(db, &prototype.AccountName{Value:name})
	if !accountWrap.CheckExist() {
		return false
	}

	newStamina := calculateNewStaminaEMA(accountWrap.GetStamina(),0,accountWrap.GetStaminaUseBlock(),now)
	maxStamina := s.calculateUserMaxStamina(db,name)
	if maxStamina < newStamina {
		return false
	}
	if maxStamina - newStamina < num {
		return false
	}
	newStamina = calculateNewStaminaEMA(newStamina,num,now,now)

	accountWrap.Modify(func(tInfo *table.SoAccount) {
		tInfo.Stamina = newStamina
		tInfo.StaminaUseBlock = now
	})
	return true
}

func (s *ResourceLimiter) ConsumeLeft(db iservices.IDatabaseRW,name string, now uint64) bool {
	accountWrap := table.NewSoAccountWrap(db, &prototype.AccountName{Value:name})
	if !accountWrap.CheckExist() {
		return false
	}

	maxStamina := s.calculateUserMaxStamina(db,name)

	accountWrap.Modify(func(tInfo *table.SoAccount) {
		tInfo.Stamina = maxStamina
		tInfo.StaminaUseBlock = now
	})
	return true
}

// FreeManager implemention
func (s *ResourceLimiter) ConsumeFree(db iservices.IDatabaseRW,name string,num uint64, now uint64) bool {
	accountWrap := table.NewSoAccountWrap(db, &prototype.AccountName{Value:name})
	if !accountWrap.CheckExist() {
		return false
	}
	newFreeStamina := calculateNewStaminaEMA(accountWrap.GetStaminaFree(),0,accountWrap.GetStaminaFreeUseBlock(),now)
	if uint64(constants.FreeStamina) - newFreeStamina < num {
		return false
	}
	newFreeStamina = calculateNewStaminaEMA(newFreeStamina,num,now,now)

	accountWrap.Modify(func(tInfo *table.SoAccount) {
		tInfo.StaminaFree = newFreeStamina
		tInfo.StaminaFreeUseBlock = now
	})
	return true
}

func (s *ResourceLimiter) GetFree(db iservices.IDatabaseRW,name string) uint64 {
	accountWrap := table.NewSoAccountWrap(db, &prototype.AccountName{Value:name})
	if !accountWrap.CheckExist() {
		return 0
	}
	return accountWrap.GetStaminaFree()
}

func (s *ResourceLimiter) GetCapacityFree() uint64 {
	return constants.FreeStamina
}

func (s *ResourceLimiter) GetStakeLeft(db iservices.IDatabaseRW,name string, now uint64) uint64 {
	accountWrap := table.NewSoAccountWrap(db, &prototype.AccountName{Value:name})
	if !accountWrap.CheckExist() {
		return 0
	}

	newStamina := calculateNewStaminaEMA(accountWrap.GetStamina(),0,accountWrap.GetStaminaUseBlock(),now)
	maxStamina := s.calculateUserMaxStamina(db,name)
	if maxStamina < newStamina {
		return 0
	}
	return maxStamina - newStamina
}

func (s *ResourceLimiter) GetFreeLeft(db iservices.IDatabaseRW,name string, now uint64) uint64 {
	accountWrap := table.NewSoAccountWrap(db, &prototype.AccountName{Value:name})
	if !accountWrap.CheckExist() {
		return 0
	}

	newStamina := calculateNewStaminaEMA(accountWrap.GetStaminaFree(),0,accountWrap.GetStaminaFreeUseBlock(),now)
	return constants.FreeStamina - newStamina
}

func (s *ResourceLimiter) ConsumeFreeLeft(db iservices.IDatabaseRW,name string, now uint64) bool {
	accountWrap := table.NewSoAccountWrap(db, &prototype.AccountName{Value:name})
	if !accountWrap.CheckExist() {
		return false
	}

	accountWrap.Modify(func(tInfo *table.SoAccount) {
		tInfo.StaminaFree = constants.FreeStamina
		tInfo.StaminaFreeUseBlock = now
	})
	return true
}

func calculateNewStamina(oldStamina uint64, useStamina uint64, lastTime uint64, now uint64) uint64 {
	blocks := uint64(constants.WindowSize)
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

func divideCeil(num,den uint64) uint64 {
	v := num / den
	if num % den > 0{
		v += 1
	}
	return v
}

func calculateNewStaminaEMA(oldStamina, useStamina uint64, lastTime uint64, now uint64) uint64 {
	blocks := uint64(constants.WindowSize)
	avgOld := divideCeil(oldStamina*constants.LimitPrecision,blocks)
	avgUse := divideCeil(useStamina*constants.LimitPrecision,blocks)
	if now > lastTime { // assert ?
		if now < lastTime + blocks {
			delta := now - lastTime
			decay := float64(blocks - delta) / float64(blocks)
			newStamina := float64(avgOld) * decay
			avgOld = uint64(newStamina)
		} else {
			avgOld = 0
		}
	}
	avgOld += avgUse
	return avgOld * constants.WindowSize / constants.LimitPrecision
}