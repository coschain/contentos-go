package utils

import (
	"fmt"
	"strconv"
	"testing"
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

func TestStakeManager_LockUpCos(t *testing.T) {
	initEvn()

	sm := NewStakeManager()
	name := "0"
	oldCos := db[name].cos
	oldVest := db[name].vest
	stakeCos := uint64(99)
	if !sm.LockUpCos(name,stakeCos) {
		t.Error(lockErr)
	}
	if db[name].cos != (oldCos - stakeCos) {
		t.Error(lockErr)
	}
	if db[name].vest != (oldVest + stakeCos) {
		t.Error(lockErr)
	}
}

func TestStakeManager_LockUpCos2(t *testing.T) {
	initEvn()

	sm := NewStakeManager()
	name := "0"
	oldCos := db[name].cos
	stakeCos := oldCos + 1
	if sm.LockUpCos(name,stakeCos) {
		t.Error(lockErr)
	}
}

func TestStakeManager_ReleaseCos(t *testing.T) {
	initEvn()

	sm := NewStakeManager()
	name := "0"
	oldCos := db[name].cos
	oldVest := db[name].vest
	stakeCos := uint64(99)
	sm.LockUpCos(name,stakeCos)
	if db[name].cos != (oldCos - stakeCos) {
		t.Error(lockErr)
	}
	if db[name].vest != (oldVest + stakeCos) {
		t.Error(lockErr)
	}

	sm.ReleaseCos(name,stakeCos)
	if db[name].cos != oldCos {
		t.Error(releaseErr)
	}
	if db[name].vest != oldVest {
		t.Error(releaseErr)
	}
}

func TestStakeManager_ReleaseCos2(t *testing.T) {
	initEvn()

	sm := NewStakeManager()
	name := "0"
	oldCos := db[name].cos
	oldVest := db[name].vest
	stakeCos := uint64(99)
	sm.LockUpCos(name,stakeCos)
	if db[name].cos != (oldCos - stakeCos) {
		t.Error(lockErr)
	}
	if db[name].vest != (oldVest + stakeCos) {
		t.Error(lockErr)
	}

	if sm.ReleaseCos(name,stakeCos+1) {
		t.Error(releaseErr)
	}
	if db[name].cos == oldCos {
		t.Error(releaseErr)
	}
	if db[name].vest == oldVest {
		t.Error(releaseErr)
	}
}

func TestStakeManager_Consume1(t *testing.T) {
	initEvn()

	sm := NewStakeManager()
	name := "0"
	stakeCos := uint64(100)

	sm.LockUpCos(name,stakeCos)

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
	step := uint64(oneDayBlocks * 0.5)
	global.addBlockNum(step)
	sm.Consume(name,1,global.getBlockNum())
	if sm.Get(name) != 51 {
		t.Error(consumeErr)
	}
	step = uint64(oneDayBlocks * 0.5)
	global.addBlockNum(step)
	sm.Consume(name,0,global.getBlockNum())
	if sm.Get(name) != 25 {
		t.Error(consumeErr)
	}
	step = uint64(oneDayBlocks)
	global.addBlockNum(step)
	sm.Consume(name,0,global.getBlockNum())
	if sm.Get(name) != 0 {
		t.Error(consumeErr)
	}
}

func TestStakeManager_Consume2(t *testing.T) {
	initEvn()

	sm := NewStakeManager()
	stakeCos := uint64(50)
	for i := 0; i < actorsNum; i++ {
		name := strconv.Itoa(i)
		sm.LockUpCos(name,stakeCos)
		sm.Consume(name,25,global.getBlockNum())
	}
	step := uint64(oneDayBlocks * 0.3)
	global.addBlockNum(step)

	for i := 0; i < actorsNum; i++ {
		name := strconv.Itoa(i)
		sm.LockUpCos(name,stakeCos)
		sm.Consume(name,0,global.getBlockNum())
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

func TestStakeManager_Consume3(t *testing.T) {
	initEvn()

	sm := NewStakeManager()
	stakeCos := uint64(1)
	name := "0"
	if !sm.LockUpCos(name,stakeCos) {
		t.Error(lockErr)
	}

	if sm.Consume(name,sm.GetCapacity(name)+1,global.getBlockNum()) {
		t.Error(consumeErr)
	}
}

// each user lock up different cos, but use same stamina, their recover should same
func TestStakeManager_Consume4(t *testing.T) {
	initEvn()
	sm := NewStakeManager()
	stakeCos := uint64(1)
	for i := 0; i < actorsNum; i++ {
		name := strconv.Itoa(i)
		if !sm.LockUpCos(name,stakeCos) {
			t.Error(lockErr)
		}
		stakeCos++
	}
	// use minimum as consume value
	consume := sm.GetCapacity("0")
	fmt.Println("minimum capacity:",consume)

	// consume
	for i := 0; i < actorsNum; i++ {
		name := strconv.Itoa(i)
		if !sm.Consume(name,consume,global.getBlockNum()) {
			t.Error(consumeErr)
		}
	}

	step := uint64(oneDayBlocks * 0.3)
	global.addBlockNum(step)

	// recover
	for i := 0; i < actorsNum; i++ {
		name := strconv.Itoa(i)
		if !sm.Consume(name,0,global.getBlockNum()) {
			t.Error(consumeErr)
		}
	}

	// check
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

	sm := NewStakeManager()
	stakeCos := uint64(1)
	for i := 0; i < actorsNum; i++ {
		name := strconv.Itoa(i)
		if !sm.LockUpCos(name,stakeCos) {
			t.Error(lockErr)
		}
	}
	// use minimum as consume value
	consume := sm.GetCapacity("0")
	fmt.Println("minimum capacity:",consume)

	// consume
	for i := 0; i < actorsNum; i++ {
		name := strconv.Itoa(i)
		if !sm.Consume(name,consume,global.getBlockNum()) {
			t.Error(consumeErr)
		}
		consume -= 10
	}

	step := uint64(oneDayBlocks * 0.3)
	global.addBlockNum(step)

	// recover
	for i := 0; i < actorsNum; i++ {
		name := strconv.Itoa(i)
		if !sm.Consume(name,0,global.getBlockNum()) {
			t.Error(consumeErr)
		}
	}

	// check
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

	sm := NewStakeManager()
	stakeCos := uint64(1)
	for i := 0; i < actorsNum; i++ {
		name := strconv.Itoa(i)
		sm.LockUpCos(name,stakeCos)
	}

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

func TestFreeManager_ConsumeFree(t *testing.T) {

}
