package iservices

const (
	Insert = iota
	Update
	Replace
	Delete
)

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
