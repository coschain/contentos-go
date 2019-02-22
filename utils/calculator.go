package utils

func calculateNewStamina(oldStamina uint64, useStamina uint64, lastTime uint64, now uint64) uint64 {
	blockSize := uint64(RECOVER_WINDOW/3)
	if now > lastTime { // assert ?
		if now < lastTime + blockSize {
			delta := now - lastTime
			decay := float64(blockSize - delta) / float64(blockSize)
			newStamina := float64(oldStamina) * decay
			oldStamina = uint64(newStamina)
		} else {
			oldStamina = 0
		}
	}
	oldStamina += useStamina
	return oldStamina
}

func calculateUserMaxStamina(name string) uint64 {
	if _,ok := db[name]; !ok {
		return 0
	}
	a,_ := db[name]
	userMax := float64(a.vest * CHAIN_STAMINA)/float64(global.totalVest)
	return uint64(userMax)
}