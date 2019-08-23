package blocklog

import "github.com/coschain/contentos-go/prototype"

type StateChange struct {
	// Type is the type of change, e.g. "account_balance"
	Type string			`json:"type"`

	// Transaction is ordinal of the transaction which made this change.
	// -1 if the change is not caused by a transaction.
	Transaction int		`json:"trx"`

	// Operation is ordinal of the operation which made this change.
	// -1 if the change is not caused by an operation.
	Operation int		`json:"op"`

	// Cause is a description about the reason of the change.
	Cause string		`json:"cause"`

	// Change is the detailed changes.
	// The actual data type depends on Type.
	Change interface{}	`json:"change"`
}

type TransactionLog struct {
	TrxId 		string							`json:"id"`
	Receipt 	*prototype.TransactionReceipt	`json:"receipt"`
	Operations 	[]*prototype.Operation			`json:"ops"`
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
}

type InternalStateChangeSlice []*internalStateChange
