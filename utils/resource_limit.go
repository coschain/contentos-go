package utils

import (
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
)

type IConsumer interface {
	Consume(name string, num uint64, now uint64) bool
	ConsumeLeft(name string, now uint64) bool
}

type IFreeConsumer interface {
	ConsumeFree(name string, num uint64, now uint64) bool
	ConsumeFreeLeft(name string, now uint64) bool
}

type IGetter interface {
	Get(name string) uint64
	GetCapacity(name string) uint64
	GetStakeLeft(name string, now uint64) uint64
}

type IFreeGetter interface {
	GetFree(name string) uint64
	GetCapacityFree() uint64
	GetFreeLeft(name string, now uint64) uint64
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
	db      iservices.IDatabaseService
}

func NewResourceLimiter(db iservices.IDatabaseService) *ResourceLimiter {
	return &ResourceLimiter{db: db}
}

const PRECISION = 10000

// recover minimum time gap
const MIN_RECOVER_DURATION = 45

// ResourceLimiter implemention
func (s *ResourceLimiter) Get(name string) uint64{
	accountWrap := table.NewSoAccountWrap(s.db, &prototype.AccountName{Value:name})
	if !accountWrap.CheckExist() {
		return 0
	}
	return accountWrap.GetStamina()
}

func (s *ResourceLimiter) GetCapacity(name string) uint64 {
	return s.calculateUserMaxStamina(name)
}

var SINGLE int32 = 1
func (s *ResourceLimiter) calculateUserMaxStamina(name string) uint64 {
	accountWrap := table.NewSoAccountWrap(s.db, &prototype.AccountName{Value:name})
	if !accountWrap.CheckExist() {
		return 0
	}
	dgpWrap := table.NewSoGlobalWrap(s.db,&SINGLE)

	vest := accountWrap.GetVestingShares().Value
	stakeVest := accountWrap.GetStakeVesting().Value

	totalVest := vest + stakeVest
	allVest := dgpWrap.GetProps().TotalVestingShares.Value
	userMax := float64( totalVest)/float64(allVest) * constants.OneDayStamina
	return uint64(userMax)
}

func (s *ResourceLimiter) Consume(name string, num uint64, now uint64) bool {
	accountWrap := table.NewSoAccountWrap(s.db, &prototype.AccountName{Value:name})
	if !accountWrap.CheckExist() {
		return false
	}

	newStamina := calculateNewStaminaEMA(accountWrap.GetStamina(),0,accountWrap.GetStaminaUseBlock(),now)
	maxStamina := s.calculateUserMaxStamina(name)
	if maxStamina - newStamina < num {
		return false
	}
	newStamina = calculateNewStaminaEMA(newStamina,num,now,now)

	accountWrap.MdStamina(newStamina)
	accountWrap.MdStaminaUseBlock(now)
	return true
}

func (s *ResourceLimiter) ConsumeLeft(name string, now uint64) bool {
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
func (s *ResourceLimiter) ConsumeFree(name string,num uint64, now uint64) bool {
	accountWrap := table.NewSoAccountWrap(s.db, &prototype.AccountName{Value:name})
	if !accountWrap.CheckExist() {
		return false
	}
	newFreeStamina := calculateNewStaminaEMA(accountWrap.GetStaminaFree(),0,accountWrap.GetStaminaFreeUseBlock(),now)
	if uint64(constants.FreeStamina) - newFreeStamina < num {
		return false
	}
	newFreeStamina = calculateNewStaminaEMA(newFreeStamina,num,now,now)

	accountWrap.MdStaminaFree(newFreeStamina)
	accountWrap.MdStaminaFreeUseBlock(now)
	return true
}

func (s *ResourceLimiter) GetFree(name string) uint64 {
	accountWrap := table.NewSoAccountWrap(s.db, &prototype.AccountName{Value:name})
	if !accountWrap.CheckExist() {
		return 0
	}
	return accountWrap.GetStaminaFree()
}

func (s *ResourceLimiter) GetCapacityFree() uint64 {
	return constants.FreeStamina
}

func (s *ResourceLimiter) GetStakeLeft(name string, now uint64) uint64 {
	accountWrap := table.NewSoAccountWrap(s.db, &prototype.AccountName{Value:name})
	if !accountWrap.CheckExist() {
		return 0
	}

	newStamina := calculateNewStaminaEMA(accountWrap.GetStamina(),0,accountWrap.GetStaminaUseBlock(),now)
	maxStamina := s.calculateUserMaxStamina(name)
	return maxStamina - newStamina
}

func (s *ResourceLimiter) GetFreeLeft(name string, now uint64) uint64 {
	accountWrap := table.NewSoAccountWrap(s.db, &prototype.AccountName{Value:name})
	if !accountWrap.CheckExist() {
		return 0
	}

	newStamina := calculateNewStaminaEMA(accountWrap.GetStaminaFree(),0,accountWrap.GetStaminaFreeUseBlock(),now)
	return constants.FreeStamina - newStamina
}

func (s *ResourceLimiter) ConsumeFreeLeft(name string, now uint64) bool {
	accountWrap := table.NewSoAccountWrap(s.db, &prototype.AccountName{Value:name})
	if !accountWrap.CheckExist() {
		return false
	}

	accountWrap.MdStaminaFree(constants.FreeStamina)
	accountWrap.MdStaminaFreeUseBlock(now)
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