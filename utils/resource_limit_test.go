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
const oneDayBlocks = 28800

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
	}
}

func TestStakeManager_Consume1(t *testing.T) {
	initEvn()

	sm := NewResourceLimiter(nil)
	name := "0"

	if sm.Get(name) != 0 {
		t.Error("init stamina error")
	}
	global.addBlockNum(10)
	if !sm.Consume(name,0,global.getBlockNum()) {
		t.Error(consumeErr)
	}
	if sm.Get(name) != 0 {
		t.Error(consumeErr)
	}
	sm.Consume(name,100,global.getBlockNum())
	if sm.Get(name) != 100 {
		t.Error(consumeErr)
	}
	// recover and consume check
	step := uint64(oneDayBlocks * 0.5)
	global.addBlockNum(step)
	sm.Consume(name,1,global.getBlockNum())
	if sm.Get(name) != 51 {
		t.Error(consumeErr)
	}
	// recover check
	step = uint64(oneDayBlocks * 0.5)
	global.addBlockNum(step)
	sm.Consume(name,0,global.getBlockNum())
	if sm.Get(name) != 25 {
		t.Error(consumeErr)
	}
	// recover all check
	step = uint64(oneDayBlocks)
	global.addBlockNum(step)
	sm.Consume(name,0,global.getBlockNum())
	if sm.Get(name) != 0 {
		t.Error(consumeErr)
	}
}

func TestStakeManager_Consume2(t *testing.T) {
	initEvn()

	sm := NewResourceLimiter(nil)
	// stake same, consume same
	for i := 0; i < actorsNum; i++ {
	}

	step := uint64(oneDayBlocks * 0.3)
	global.addBlockNum(step)

	// recover same
	for i := 0; i < actorsNum; i++ {
	}

	for i := 0; i < actorsNum; i++ {
		if i == actorsNum - 1 {
			break
		}
		name := strconv.Itoa(i)
		nameNext := strconv.Itoa(i+1)
		if sm.Get(name) != sm.Get(nameNext) {
			t.Error(consumeErr)
		}
	}
}

// each user lock up different cos, but use same stamina, their recover should same
func TestStakeManager_Consume4(t *testing.T) {
	initEvn()
	sm := NewResourceLimiter(nil)
	// stake different

	//
	consume := sm.GetCapacity("0")
	fmt.Println("minimum capacity:",consume)

	// consume same
	for i := 0; i < actorsNum; i++ {
		name := strconv.Itoa(i)
		if !sm.Consume(name,consume,global.getBlockNum()) {
			t.Error(consumeErr)
		}
	}

	step := uint64(oneDayBlocks * 0.3)
	global.addBlockNum(step)

	// recover same
	for i := 0; i < actorsNum; i++ {
		name := strconv.Itoa(i)
		if !sm.Consume(name,0,global.getBlockNum()) {
			t.Error(consumeErr)
		}
	}

	// each should same
	for i := 0; i < actorsNum; i++ {
		if i == actorsNum - 1 {
			break
		}
		name := strconv.Itoa(i)
		nameNext := strconv.Itoa(i+1)
		if sm.Get(name) != sm.Get(nameNext) {
			t.Error(consumeErr)
		}
	}
}

// each user lock up same cos, but use different stamina, their recover not same
func TestStakeManager_Consume5(t *testing.T) {
	initEvn()

	sm := NewResourceLimiter(nil)

	// stake same

	// use minimum as consume value
	consume := sm.GetCapacity("0")
	fmt.Println("minimum capacity:",consume)

	// consume different
	for i := 0; i < actorsNum; i++ {
		name := strconv.Itoa(i)
		if !sm.Consume(name,consume,global.getBlockNum()) {
			t.Error(consumeErr)
		}
		consume -= 10
	}

	step := uint64(oneDayBlocks * 0.3)
	global.addBlockNum(step)

	// recover same
	for i := 0; i < actorsNum; i++ {
		name := strconv.Itoa(i)
		if !sm.Consume(name,0,global.getBlockNum()) {
			t.Error(consumeErr)
		}
	}

	// each should different
	for i := 0; i < actorsNum; i++ {
		if i == actorsNum - 1 {
			break
		}
		name := strconv.Itoa(i)
		nameNext := strconv.Itoa(i+1)
		if sm.Get(name) == sm.Get(nameNext) {
			t.Error(consumeErr)
		}
	}
}

func TestStakeManager_GetCapacity(t *testing.T) {
	initEvn()

	sm := NewResourceLimiter(nil)
	// stake same

	// capacity should same
	for i := 0; i < actorsNum; i++ {
		if i == actorsNum - 1 {
			break
		}
		name := strconv.Itoa(i)
		nameNext := strconv.Itoa(i+1)
		if sm.GetCapacity(name) != sm.GetCapacity(nameNext) {
			t.Error(consumeErr)
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

	avg = 1
	for i:= startBlock; i < endBlock; i++ {
		avg = calculateNewStamina(avg,data[i],i-1,i)
		if i < 10 {
			println("trace:",avg)
		}
	}
	fmt.Println("avg:",avg/constants.WindowSize," all:",avg)
}
