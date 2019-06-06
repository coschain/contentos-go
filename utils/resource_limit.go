package utils

import (
	"github.com/coschain/contentos-go/common/constants"
	"math/big"
)

type IConsumer interface {
	Consume(oldResource, num, preTime, now, maxStamina uint64) (bool,uint64)
}

type IFreeConsumer interface {
	ConsumeFree(maxFreeStamina, oldResource,num, preTime, now uint64) (bool,uint64)
}

type IGetter interface {
	GetStakeLeft(oldResource,preTime,now,maxStamina uint64) (string,uint64)
}

type IFreeGetter interface {
	GetFreeLeft(maxFreeStamina, oldResource, preTime, now uint64) (string,uint64)
}

type ITpsUpdater interface {
	UpdateDynamicStamina(tpsInWindow,oneDayStamina,trxCount,lastUpdate,blockNum,expectedTps uint64) (uint64,uint64)
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
func (s *ResourceLimiter) ConsumeFree(maxFreeStamina,oldResource uint64,num uint64, preTime, now uint64) (bool,uint64) {
	newFreeStamina := calculateNewStaminaEMA(oldResource,0,preTime,now)
	if maxFreeStamina < newFreeStamina {
		return false,0
	}
	if maxFreeStamina - newFreeStamina < num {
		return false,0
	}
	newFreeStamina = calculateNewStaminaEMA(newFreeStamina,num,now,now)
	return true,newFreeStamina
}

func (s *ResourceLimiter) GetStakeLeft(oldResource,preTime,now,maxStamina uint64) (string,uint64) {
	newStamina := calculateNewStaminaEMA( oldResource,0, preTime,now)
	if maxStamina < newStamina {
		return constants.StakeStaminaOverFlow,0
	}
	return "",maxStamina - newStamina
}

func (s *ResourceLimiter) GetFreeLeft(maxFreeStamina, oldResource, preTime, now uint64) (string, uint64) {
	newStamina := calculateNewStaminaEMA(oldResource,0,preTime,now)
	if maxFreeStamina < newStamina {
		return constants.FreeStaminaOverFlow,0
	}
	return "",maxFreeStamina - newStamina
}

func calculateNewStaminaEMA(oldStamina, useStamina, lastTime, now uint64) uint64 {
	return calculateEMA(oldStamina, useStamina, lastTime, now, constants.WindowSize)
}

func (s *ResourceLimiter) UpdateDynamicStamina(tpsInWindow,oneDayStamina,trxCount,lastUpdate,blockNum,expectedTps uint64) (uint64,uint64) {
	tpsInWindowNew := calculateTpsEMA(tpsInWindow,trxCount,lastUpdate,blockNum)
	return updateDynamicOneDayStamina(oneDayStamina,tpsInWindowNew/constants.TpsWindowSize,expectedTps),tpsInWindowNew
}

func calculateTpsEMA(oldTrxs, newTrxs, lastTime, now uint64) uint64 {
	return calculateEMA(oldTrxs, newTrxs, lastTime, now, constants.TpsWindowSize)
}

func updateDynamicOneDayStamina(oldOneDayStamina, avgTps,expectedTps uint64) uint64 {
	change := oldOneDayStamina / 100
	if avgTps > expectedTps {
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

func divideCeilBig(num,den *big.Int) *big.Int {
	tmp := new(big.Int)
	tmp.Div(num, den)
	if num.Mod(num, den).Uint64() > 0 {
		tmp.Add(tmp, big.NewInt(1))
	}
	return tmp
}

func calculateEMA(oldTrxs, newTrxs uint64, lastTime uint64, now, period uint64) uint64 {
	blocks := big.NewInt(int64(period))
	precisionBig := big.NewInt(constants.LimitPrecision)
	oldTrxsBig := big.NewInt(int64(oldTrxs))
	oldTrxsBig.Mul(oldTrxsBig, precisionBig)

	newTrxsBig := big.NewInt(int64(newTrxs))
	newTrxsBig.Mul(newTrxsBig, precisionBig)

	avgOld := divideCeilBig(oldTrxsBig, blocks)
	avgUse := divideCeilBig(newTrxsBig, blocks)
	if now > lastTime { // assert ?
		if now < lastTime+blocks.Uint64() {
			delta := now - lastTime
			gap := big.NewInt(int64(blocks.Uint64() - delta))
			gap.Mul(gap, precisionBig)
			decay := divideCeilBig(gap, blocks)

			avgOld.Mul(avgOld, decay)
			avgOld.Div(avgOld, precisionBig)
		} else {
			avgOld.SetUint64(0)
		}
	}
	avgOld.Add(avgOld,avgUse)
	avgOld.Mul(avgOld, blocks).Div(avgOld, precisionBig)
	return avgOld.Uint64()
}