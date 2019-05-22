package utils

import (
	"fmt"
	"github.com/coschain/contentos-go/common/constants"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

var lockErr = "lock error"
var releaseErr = "release error"
var consumeErr = "consume error"

const actorsNum = 100
const initCos = 100
const oneDayBlocks = constants.WindowSize

func initEvn() {
	initDB()
	initGlobal()
	addActors()
}

func addActors() {
	// add test accounts
	for i :=0; i<actorsNum;i++ {
		name := strconv.Itoa(i)
		db[name] = &account{name:name,cos:initCos,
		staminaUseTime:global.getBlockNum(),staminaFreeUseTime:global.getBlockNum()}
		db[name].vest = 1
	}
	global.totalVest = actorsNum
}

func TestStakeManager_Consume1(t *testing.T) {
	initEvn()

	sm := NewResourceLimiter()
	name := "0"

	if db[name].stamina != 0 {
		t.Error("init stamina error")
	}
	global.addBlockNum(10)
	if ok,_ := sm.Consume(db[name].stamina,0,db[name].staminaUseTime,global.getBlockNum(),maxStakeStamina(name));!ok {
		t.Error(consumeErr)
	}
	if db[name].stamina != 0 {
		t.Error(consumeErr)
	}
	_,c := sm.Consume(db[name].stamina,100,db[name].staminaUseTime,global.getBlockNum(),maxStakeStamina(name))
	if c != 100 {
		t.Error(consumeErr,c)
		return
	}
	db[name].stamina = c
	db[name].staminaUseTime = global.getBlockNum()

	// recover and consume check
	step := uint64(oneDayBlocks * 0.5)
	global.addBlockNum(step)
	_,c = sm.Consume(db[name].stamina,1,db[name].staminaUseTime,global.getBlockNum(),maxStakeStamina(name))
	if c != 51 {
		t.Error(consumeErr,c)
		return
	}
	db[name].stamina = c
	db[name].staminaUseTime = global.getBlockNum()

	// recover check
	step = uint64(oneDayBlocks * 0.5)
	global.addBlockNum(step)
	_,c = sm.Consume(db[name].stamina,0,db[name].staminaUseTime,global.getBlockNum(),maxStakeStamina(name))
	if c != 25 {
		t.Error(consumeErr,c)
		return
	}
	db[name].stamina = c
	db[name].staminaUseTime = global.getBlockNum()

	// recover all check
	step = uint64(oneDayBlocks)
	global.addBlockNum(step)
	_,c = sm.Consume(db[name].stamina,0,db[name].staminaUseTime,global.getBlockNum(),maxStakeStamina(name))
	if c != 0 {
		t.Error(consumeErr)
	}
}

func TestStakeManager_Consume2(t *testing.T) {
	initEvn()

	sm := NewResourceLimiter()
	// stake same, consume same
	global.totalVest = actorsNum
	for i := 0; i < actorsNum; i++ {
		name := strconv.Itoa(i)
		db[name].vest = 1
		ok,c := sm.Consume(db[name].stamina,100000,db[name].staminaUseTime,global.getBlockNum(),maxStakeStamina(name))
		if !ok {
			t.Error(consumeErr," ",name," ",c)
			return
		}
		db[name].stamina = c
		db[name].staminaUseTime = global.getBlockNum()
	}

	step := uint64(oneDayBlocks * 0.3)
	global.addBlockNum(step)

	// recover same
	for i := 0; i < actorsNum; i++ {
		name := strconv.Itoa(i)
		db[name].vest = 1
		ok,c := sm.Consume(db[name].stamina,0,db[name].staminaUseTime,global.getBlockNum(),maxStakeStamina(name))
		if !ok {
			t.Error(consumeErr," ",name," ",c)
			return
		}
		db[name].stamina = c
		db[name].staminaUseTime = global.getBlockNum()
	}

	for i := 0; i < actorsNum; i++ {
		if i == actorsNum - 1 {
			break
		}
		name := strconv.Itoa(i)
		nameNext := strconv.Itoa(i+1)
		if db[name].stamina != db[nameNext].stamina {
			t.Error(consumeErr)
		}
		//fmt.Println("1:",db[name].stamina," 2:",db[nameNext].stamina)
	}
}

// each user lock up different cos, but use same stamina, their recover should same
func TestStakeManager_Consume4(t *testing.T) {
	initEvn()
	sm := NewResourceLimiter()
	// stake different
	var start uint64 = 1
	var sum uint64 = 0
	for i := 0; i < actorsNum; i++ {
		name := strconv.Itoa(i)
		db[name].vest = start
		start++
		sum += start
		//fmt.Println(db[name].vest)
	}
	global.totalVest = sum

	//
	consume := maxStakeStamina("0")
	fmt.Println("minimum stamina capacity:",consume)

	// consume same
	for i := 0; i < actorsNum; i++ {
		name := strconv.Itoa(i)
		ok,c:=sm.Consume(db[name].stamina,consume,db[name].staminaUseTime,global.getBlockNum(),maxStakeStamina(name));
		if !ok {
			t.Error(consumeErr)
			return
		}
		db[name].stamina = c
		db[name].staminaUseTime = global.getBlockNum()
	}

	step := uint64(oneDayBlocks * 0.3)
	global.addBlockNum(step)

	// recover same
	for i := 0; i < actorsNum; i++ {
		name := strconv.Itoa(i)
		ok,c := sm.Consume(db[name].stamina,0,db[name].staminaUseTime,global.getBlockNum(),maxStakeStamina(name))
		if !ok {
			t.Error(consumeErr," ",name," ",c)
			return
		}
		//fmt.Println("name:",db[name]," stamina:",c)
		db[name].stamina = c
		db[name].staminaUseTime = global.getBlockNum()
	}

	// each should same
	for i := 0; i < actorsNum; i++ {
		if i == actorsNum - 1 {
			break
		}
		name := strconv.Itoa(i)
		nameNext := strconv.Itoa(i+1)
		if db[name].stamina != db[nameNext].stamina {
			t.Error(consumeErr)
		}
		//fmt.Println("1:",db[name].stamina," 2:",db[nameNext].stamina)
	}
}

// each user lock up same cos, but use different stamina, their recover not same
func TestStakeManager_Consume5(t *testing.T) {
	initEvn()

	sm := NewResourceLimiter()

	// stake same

	// use minimum as consume value
	consume := maxStakeStamina("0")
	fmt.Println("minimum capacity:",consume)

	// consume different
	for i := 0; i < actorsNum; i++ {
		name := strconv.Itoa(i)
		ok,c := sm.Consume(db[name].stamina,consume,db[name].staminaUseTime,global.getBlockNum(),maxStakeStamina(name))
		if !ok {
			t.Error(consumeErr," ",name)
			return
		}
		db[name].stamina = c
		db[name].staminaUseTime = global.getBlockNum()
		consume -= 10
	}

	step := uint64(oneDayBlocks * 0.3)
	global.addBlockNum(step)

	// recover same
	for i := 0; i < actorsNum; i++ {
		name := strconv.Itoa(i)
		ok,c := sm.Consume(db[name].stamina,0,db[name].staminaUseTime,global.getBlockNum(),maxStakeStamina(name))
		if !ok {
			t.Error(consumeErr," ",name)
			return
		}
		db[name].stamina = c
		db[name].staminaUseTime = global.getBlockNum()
	}

	// each should different
	for i := 0; i < actorsNum; i++ {
		if i == actorsNum - 1 {
			break
		}
		name := strconv.Itoa(i)
		nameNext := strconv.Itoa(i+1)
		if db[name].stamina == db[nameNext].stamina {
			t.Error(consumeErr,"name:",name," ",db[name].stamina," ","name2:",nameNext," ",db[nameNext].stamina)
		}
	}
}

func TestStakeManager_GetCapacity(t *testing.T) {
	initEvn()
	// stake same

	// capacity should same
	for i := 0; i < actorsNum; i++ {
		if i == actorsNum - 1 {
			break
		}
		name := strconv.Itoa(i)
		nameNext := strconv.Itoa(i+1)
		if maxStakeStamina(name) != maxStakeStamina(nameNext) {
			t.Error(consumeErr)
		}
		//fmt.Println("1:",maxStakeStamina(name)," 2:",maxStakeStamina(nameNext))
	}
}

func TestDynamicTps(t *testing.T) {
	initEvn()

	var trxsInWindow,trxsInBlock uint64 = 0,100
	preTime := global.blockNum
	now := preTime
	for i:=0;i<3000;i++ {
		now++
		trxsInWindowNew := calculateTpsEMA(trxsInWindow,trxsInBlock,preTime,now)
		preTime=now
		fmt.Println("trxs in window:",trxsInWindowNew," trxsInBlock:",trxsInBlock," block num:",now, " real tps:",trxsInWindowNew/constants.TpsWindowSize)
		//trxsInBlock += 1
		trxsInWindow = trxsInWindowNew
		if i > 1000 && i < 2000 {
			if trxsInBlock > 0 {
				trxsInBlock--
			}
		}
		if i >= 2000 {
			trxsInBlock++
		}
	}
}

func TestDynamicStamina(t *testing.T) {
	initEvn()

	var trxsInWindow,trxsInBlock,oneDayStamina uint64 = 0,1000,constants.OneDayStamina
	preTime := global.blockNum
	now := preTime
	for i:=0;i<3000;i++ {
		now++
		trxsInWindowNew := calculateTpsEMA(trxsInWindow,trxsInBlock,preTime,now)
		preTime = now
		oneDayStaminaNew := updateDynamicOneDayStamina(oneDayStamina,trxsInWindowNew/constants.TpsWindowSize)
		fmt.Println("oneDayStamina:",oneDayStaminaNew,"trxs in window:",trxsInWindowNew," trxsInBlock:",trxsInBlock," block num:",now, " real tps:",trxsInWindowNew/constants.TpsWindowSize)
		trxsInWindow = trxsInWindowNew
		oneDayStamina = oneDayStaminaNew

		if i > 1000 && i < 2000 {
			if trxsInBlock > 10 {
				trxsInBlock-=10
			}
		}
		if i >= 2000 {
			trxsInBlock+=10
		}

	}
}

func Test_EMA(t *testing.T) {

	var startBlock uint64 = 1
	var endBlock uint64 = 86400
	var avg uint64= 1

	rand.Seed(time.Now().UnixNano())

	data := []uint64{}
	for i:= startBlock; i <= endBlock; i++ {
		data = append(data,uint64(rand.Intn(100000000)))
	}

	for i := startBlock; i < endBlock;i++ {
		avg = calculateNewStaminaEMA(avg,data[i],i-1,i)
		if i < 10 {
			println("new EMA:",avg)
		}
	}
	fmt.Println("new EMA avg:",avg/constants.WindowSize,"new EMA all:",avg)

}
