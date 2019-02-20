package consensus

import (
	"encoding/binary"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/db/storage"
	"github.com/coschain/gobft/message"
)

type BFTCheckPoint struct {
	sabft   *SABFT
	dataDir string
	db      storage.Database

	interval uint64
	lastCP   uint64
	nextCP   *message.Commit
}

func NewBFTCheckPoint(dir string, sabft *SABFT) *BFTCheckPoint {
	db, err := storage.NewLevelDatabase(dir)
	if err != nil {
		panic(err)
	}
	return &BFTCheckPoint{
		sabft:    sabft,
		dataDir:  dir,
		db:       db,
		interval: 5,
		lastCP:   0,
		nextCP:   nil,
	}
}

func (cp *BFTCheckPoint) Make(commit *message.Commit) error {
	blockID := &common.BlockID{
		Data: commit.ProposedData,
	}
	blockNum := blockID.BlockNum()
	if blockNum-cp.lastCP < cp.interval {
		return nil
	}
	key := make([]byte, 8)
	binary.LittleEndian.PutUint64(key, blockNum)
	err := cp.db.Put(key, commit.Bytes())
	if err != nil {
		return err
	}
	cp.lastCP = blockNum
	cp.sabft.log.Info("checkpoint made at block height ", blockNum)
	return nil
}

func (cp *BFTCheckPoint) AcceptCheckPoint(commit *message.Commit) {
	//cp.sabft.log.Info("adding checkpoint/////", commit.ProposedData)
	if err := commit.ValidateBasic(); err != nil {
		cp.sabft.log.Error(err)
		return
	}
	blockID := &common.BlockID{
		Data: commit.ProposedData,
	}

	blockNum := blockID.BlockNum()
	if blockNum < cp.lastCP || !cp.sabft.ForkDB.Empty() && blockNum <= cp.sabft.ForkDB.Head().Id().BlockNum() {
		return
	}

	var nextBlockNum uint64
	if cp.nextCP != nil {
		nextBlockNum = common.BlockID{
			Data: cp.nextCP.ProposedData,
		}.BlockNum()
	}

	if nextBlockNum == 0 || blockNum < nextBlockNum {
		cp.nextCP = commit
		cp.sabft.log.Infof("next checkpoint at block height %d received.", blockNum)
	}
}

func (cp *BFTCheckPoint) GetNext(blockNum uint64) (*message.Commit, error) {
	key := make([]byte, 8)
	binary.LittleEndian.PutUint64(key, blockNum+1)
	it := cp.db.NewIterator(key, nil)
	it.Next()
	val, err := it.Value()
	if err != nil {
		return nil, err
	}
	commit, err := message.DecodeConsensusMsg(val)
	if err != nil {
		return nil, err
	}
	return commit.(*message.Commit), nil
}

func (cp *BFTCheckPoint) ValidateAndCommit(block common.ISignedBlock) bool {
	if cp.nextCP == nil || block == nil {
		return true
	}
	nextBlockID := common.BlockID{
		Data: cp.nextCP.ProposedData,
	}
	if nextBlockID != block.Id() {
		return true
	}

	// check +2/3
	if len(cp.sabft.validators)*2/3 > len(cp.nextCP.Precommits) {
		return false
	}

	if !cp.sabft.verifyCommitSig(cp.nextCP) {
		return false
	}
	cp.sabft.commit(cp.nextCP)
	cp.lastCP = nextBlockID.BlockNum()
	cp.nextCP = nil
	cp.sabft.log.Infof("checkpoint at block height %v validated.", nextBlockID.BlockNum())
	return true
}
