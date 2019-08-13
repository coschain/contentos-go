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
	TransferToStakeVest(value *prototype.Coin)
	TransferFromStakeVest(value *prototype.Vest)
	ModifyProps(modifier func(oldProps *prototype.DynamicProperties))
	UpdateTicketIncomeAndNum(income *prototype.Vest, count uint64)
}

type IGlobalPropRW interface {
	IGlobalPropReader
	IGlobalPropWriter
}

type ITrxPool interface {
	IGlobalPropRW

	PushTrx(trx *prototype.SignedTransaction) *prototype.TransactionReceiptWithInfo
	PushBlock(blk *prototype.SignedBlock, skip prototype.SkipFlag) error
	GenerateBlock(bpName string, pre *prototype.Sha256, timestamp uint32, priKey *prototype.PrivateKeyType, skip prototype.SkipFlag) (*prototype.SignedBlock, error)
	GetBlockProducerTopN(n uint32) ([]string, []*prototype.PublicKeyType)
	GetSigningPubKey(bpName string) *prototype.PublicKeyType
	SetShuffledBpList(names []string, keys []*prototype.PublicKeyType)
	GetShuffledBpList() ([]string, []*prototype.PublicKeyType)
	SetShuffle(s common.ShuffleFunc)
	// PreShuffle() must be called to notify ITrxPool that block producers shuffle is about to happen
	PreShuffle() error
	// PopBlock() rollbacks the state db to the moment just before applying block @num.
	PopBlock(num uint64) error
	// Commit() finalizes block @num.
	Commit(num uint64)

	// put trx into pending directly, no return value, so should be used by witness node to collect p2p trx
	PushTrxToPending(trx *prototype.SignedTransaction) error
	GenerateAndApplyBlock(bpName string, pre *prototype.Sha256, timestamp uint32, priKey *prototype.PrivateKeyType, skip prototype.SkipFlag) (*prototype.SignedBlock, error)
	VerifySig(name *prototype.AccountName, digest []byte, sig []byte) bool
	ValidateAddress(name string, pubKey *prototype.PublicKeyType) bool
	Sign(priv *prototype.PrivateKeyType, digest []byte) []byte
	//Fetch the latest pushed block number
	GetLastPushedBlockNum() (uint64,error)
	//Sync commit blocks to db
	SyncCommittedBlockToDB(blk common.ISignedBlock) error
	//Sync pushed blocks to DB
	SyncPushedBlocksToDB(blkList []common.ISignedBlock) error

	CalculateUserMaxStamina(db IDatabaseRW,name string) uint64
	CheckNetForRPC(name string, db IDatabaseRW, sizeInBytes uint64) (bool,uint64,uint64)
	EstimateStamina(trx *prototype.SignedTransaction) *prototype.TransactionReceiptWithInfo
}
