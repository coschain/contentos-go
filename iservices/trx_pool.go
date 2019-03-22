package iservices

import (
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/prototype"
)

//
// This file defines interfaces of Database service.
//

var TxPoolServerName = "ctrl"

type IGlobalPropReader interface {
	GetProps() *prototype.DynamicProperties
	HeadBlockTime() *prototype.TimePointSec
}

type IGlobalPropWriter interface {
	TransferToVest(value *prototype.Coin)
	TransferFromVest(value *prototype.Vest)
	AddWeightedVP(value uint64)
	ModifyProps(modifier func(oldProps *prototype.DynamicProperties))
}

type IGlobalPropRW interface {
	IGlobalPropReader
	IGlobalPropWriter
}

type ITrxPool interface {
	IGlobalPropRW

	PushTrx(trx *prototype.SignedTransaction) *prototype.TransactionReceiptWithInfo
	PushBlock(blk *prototype.SignedBlock, skip prototype.SkipFlag) error
	GenerateBlock(witness string, pre *prototype.Sha256, timestamp uint32, priKey *prototype.PrivateKeyType, skip prototype.SkipFlag) (*prototype.SignedBlock, error)
	GetWitnessTopN(n uint32) []string
	SetShuffledWitness(names []string)
	GetShuffledWitness() []string
	SetShuffle(s common.ShuffleFunc)
	// PopBlock() rollbacks the state db to the moment just before applying block @num.
	PopBlock(num uint64)
	// Commit() finalizes block @num.
	Commit(num uint64)

	// put trx into pending directly, no return value, so should be used by witness node to collect p2p trx
	PushTrxToPending(trx *prototype.SignedTransaction) error
	GenerateAndApplyBlock(witness string, pre *prototype.Sha256, timestamp uint32, priKey *prototype.PrivateKeyType, skip prototype.SkipFlag) (*prototype.SignedBlock, error)
	VerifySig(name *prototype.AccountName, digest []byte, sig []byte) bool
	ValidateAddress(name string, pubKey *prototype.PublicKeyType) bool
	Sign(priv *prototype.PrivateKeyType, digest []byte) []byte
	//Fetch the latest commit block number
	GetCommitBlockNum() (uint64,error)
	//Sync commit blocks to db
	SyncCommittedBlockToDB(blk common.ISignedBlock) error
	//Sync pushed blocks to DB
	SyncPushedBlocksToDB(blkList []common.ISignedBlock) error
}
