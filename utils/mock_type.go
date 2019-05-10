package utils

const BLOCK_INTERVAL = 3

var db map[string]*account
var global *gloabalProperty

func initDB(){
	db = make(map[string]*account)
}

func initGlobal() {
	global = &gloabalProperty{blockNum:0}
}

type account struct {
	name string
	cos uint64
	vest uint64

	// resource
	staminaCapacity uint64
	stamina uint64
	staminaFreeCapacity uint64
	staminaFree uint64
	staminaUseTime uint64
	staminaFreeUseTime uint64
}

type gloabalProperty struct {
	totalCos uint64
	totalVest uint64
	blockNum uint64
}

func (g *gloabalProperty) getBlockNum() uint64 {
	return g.blockNum
}

func (g *gloabalProperty) addBlockNum(n uint64) {
	g.blockNum += n
}