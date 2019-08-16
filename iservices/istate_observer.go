package iservices

const (
	Insert = iota
	Update
	Replace
	Delete
	Add
	Sub
)
type OpLog struct {
	Action int
	Property string
	Target string
	Result interface{}
}

type TrxLog struct {
	TrxId string
	OpLogs []OpLog
}

type BlockLog struct {
	BlockHeight uint64
	BlockId string
	BlockTime uint64
	TrxLogs []TrxLog
	Index int // the index of item in the heap
}

type IStateObserver interface {
	BeginBlock(blockNum uint64)
	NewTrxObserver() ITrxObserver
	EndBlock(blockId string)
}

type ITrxObserver interface {
	BeginTrx(trxId string)
	// action: insert, update, replace, delete
	// property: which property modified?
	// target: who modified
	// result: what became
	AddOpState(action int, property string, target string, result interface{})
	EndTrx(keep bool)
}
