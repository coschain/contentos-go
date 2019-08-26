package blocklog

import "github.com/coschain/contentos-go/prototype"

const (
	ChangeKindCreate = "create"
	ChangeKindUpdate = "update"
	ChangeKindDelete = "delete"
)

type GenericChange struct {
	Id  interface{}		`json:"id"`
	Before interface{}	`json:"before"`
	After interface{}	`json:"after"`
}

type StateChange struct {
	What string							`json:"what"`
	Kind string							`json:"kind"`
	Cause string						`json:"cause"`
	CauseExtra map[string]interface{}	`json:"cause_extra"`
	Change *GenericChange				`json:"change"`
}

type OperationData struct {
	Type	string							`json:"type"`
	Data	prototype.BaseOperation			`json:"data"`
}

type OperationLog struct {
	Op 		*OperationData					`json:"op"`
	Changes	[]*StateChange					`json:"changes"`
}

type TransactionLog struct {
	TrxId 		string							`json:"id"`
	Receipt 	*prototype.TransactionReceipt	`json:"receipt"`
	Operations 	[]*OperationLog					`json:"ops"`
}

type BlockLog struct {
	BlockId  		string					`json:"id"`
	BlockNum 		uint64					`json:"num"`
	BlockTime 		uint32					`json:"time"`
	Transactions 	[]*TransactionLog		`json:"trxs"`
	Changes     	[]*StateChange			`json:"changes"`
}

type internalStateChange struct {
	StateChange
	TransactionId string
	Transaction int
	Operation int
}

type InternalStateChangeSlice []*internalStateChange
