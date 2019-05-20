package utils

import (
	"github.com/coschain/contentos-go/common/constants"
)

type IConsumer interface {
	Consume(oldResource uint64, num uint64, preTime, now, maxStamina uint64) (bool,uint64)
}

type IFreeConsumer interface {
	ConsumeFree(oldResource uint64,num uint64, preTime, now uint64) (bool,uint64)
}

type IGetter interface {
	GetStakeLeft(oldResource,preTime,now,maxStamina uint64) uint64
}

type IFreeGetter interface {
	GetCapacityFree() uint64
	GetFreeLeft(oldResource uint64, preTime, now uint64) uint64
}

type ITpsUpdater interface {
	UpdateDynamicStamina(tpsInWindow,oneDayStamina,trxCount,lastUpdate,blockNum uint64) uint64
}

// stake resource interface
type IResourceLimiter interface {
	IConsumer
	IFreeConsumer
	IGetter
	IFreeGetter
	ITpsUpdater
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

func (s *ResourceLimiter) Consume(oldResource uint64, num uint64, preTime, now, maxStamina uint64) (bool,uint64) {
	newStamina := calculateNewStaminaEMA(oldResource,0, preTime,now)
	if maxStamina < newStamina {
		return false,0
	}
	if maxStamina - newStamina < num {
		return false,0
	}
	newStamina = calculateNewStaminaEMA(newStamina,num,now,now)
	return true,newStamina
}

// FreeManager implemention
func (s *ResourceLimiter) ConsumeFree(oldResource uint64,num uint64, preTime, now uint64) (bool,uint64) {
	newFreeStamina := calculateNewStaminaEMA(oldResource,0,preTime,now)
	if uint64(constants.FreeStamina) - newFreeStamina < num {
		return false,0
	}
	newFreeStamina = calculateNewStaminaEMA(newFreeStamina,num,now,now)
	return true,newFreeStamina
}

func (s *ResourceLimiter) GetCapacityFree() uint64 {
	return constants.FreeStamina
}

func (s *ResourceLimiter) GetStakeLeft(oldResource,preTime,now,maxStamina uint64) uint64 {
	newStamina := calculateNewStaminaEMA( oldResource,0, preTime,now)
	if maxStamina < newStamina {
		return 0
	}
	return maxStamina - newStamina
}

func (s *ResourceLimiter) GetFreeLeft(oldResource uint64, preTime, now uint64) uint64 {
	newStamina := calculateNewStaminaEMA(oldResource,0,preTime,now)
	return constants.FreeStamina - newStamina
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

func calculateNewStaminaEMA(oldStamina, useStamina, lastTime, now uint64) uint64 {
	return calculateEMA(oldStamina, useStamina, lastTime, now, constants.WindowSize)
}

func (s *ResourceLimiter) UpdateDynamicStamina(tpsInWindow,oneDayStamina,trxCount,lastUpdate,blockNum uint64) uint64 {
	tpsInWindowNew := calculateTpsEMA(tpsInWindow,trxCount,lastUpdate,blockNum)
	return updateDynamicOneDayStamina(oneDayStamina,tpsInWindowNew/constants.TpsWindowSize)
}

func calculateEMA(oldTrxs, newTrxs uint64, lastTime uint64, now, period uint64) uint64 {
	blocks := period
	avgOld := divideCeil(oldTrxs*constants.LimitPrecision,blocks)
	avgUse := divideCeil(newTrxs*constants.LimitPrecision,blocks)
	if now > lastTime { // assert ?
		if now < lastTime + blocks {
			delta := now - lastTime
			decay := float64(blocks - delta) / float64(blocks)
			tmp := float64(avgOld) * decay
			avgOld = uint64(tmp)
		} else {
			avgOld = 0
		}
	}
	avgOld += avgUse
	return avgOld * period / constants.LimitPrecision
}

func calculateTpsEMA(oldTrxs, newTrxs, lastTime, now uint64) uint64 {
	return calculateEMA(oldTrxs, newTrxs, lastTime, now, constants.TpsWindowSize)
}

func updateDynamicOneDayStamina(oldOneDayStamina, avgTps uint64) uint64 {
	change := oldOneDayStamina / 100
	if avgTps > constants.TpsExpected {
		oldOneDayStamina = oldOneDayStamina - change
	} else {
		// todo calculate user's avg stamina, if is large enough, do not expand
		oldOneDayStamina = oldOneDayStamina + change
	}
	if oldOneDayStamina < constants.OneDayStamina {
		oldOneDayStamina = constants.OneDayStamina
	}
	if oldOneDayStamina > constants.OneDayStamina * 100 {
		oldOneDayStamina = constants.OneDayStamina * 100
	}
	return oldOneDayStamina
}