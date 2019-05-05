package consensus

import (
	"encoding/binary"
	"fmt"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/db/storage"
	"github.com/coschain/gobft/message"
)

type BFTCheckPoint struct {
	sabft   *SABFT
	dataDir string
	db      storage.Database

	lastCP uint64
	cache  map[uint64]*message.Commit
	nextCP *message.Commit
}

func NewBFTCheckPoint(dir string, sabft *SABFT) *BFTCheckPoint {
	db, err := storage.NewLevelDatabase(dir)
	if err != nil {
		panic(err)
	}
	lc := sabft.ForkDB.LastCommitted().BlockNum()
	return &BFTCheckPoint{
		sabft:   sabft,
		dataDir: dir,
		db:      db,
		lastCP:  lc,
		nextCP:  nil,
		cache:   make(map[uint64]*message.Commit),
	}
}

func (cp *BFTCheckPoint) Make(commit *message.Commit) error {
	if err := commit.ValidateBasic(); err != nil {
		cp.sabft.log.Error(err)
		return err
	}
	blockID := &common.BlockID{
		Data: commit.ProposedData,
	}
	blockNum := blockID.BlockNum()
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, blockNum)
	err := cp.db.Put(key, commit.Bytes())
	if err != nil {
		return err
	}
	cp.lastCP = blockNum
	cp.sabft.log.Info("checkpoint made at block height ", blockNum)
	return nil
}

func (cp *BFTCheckPoint) Add(commit *message.Commit) {
	//cp.sabft.log.Info("adding checkpoint/////", commit.ProposedData)
	if err := commit.ValidateBasic(); err != nil {
		cp.sabft.log.Error(err)
		return
	}
	blockID := &common.BlockID{
		Data: commit.ProposedData,
	}

	blockNum := blockID.BlockNum()
	if blockNum <= cp.lastCP || blockNum >= cp.lastCP+constants.MaxUncommittedBlockNum {
		return
	}

	cp.cache[blockNum] = commit
}

func (cp *BFTCheckPoint) Commit(num uint64) {
	cp.lastCP = num
	for k := range cp.cache {
		if k <= num {
			delete(cp.cache, k)
		}
	}
}

func (cp *BFTCheckPoint) GetNext(blockNum uint64) (*message.Commit, error) {
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, blockNum+1)
	var val []byte
	cp.db.Iterate(key, nil, false, func(key, value []byte) bool {
		val = common.CopyBytes(value)
		return false
	})
	if len(val) == 0 {
		return nil, fmt.Errorf("BFTCheckPoint.GetNext(%d) not found", blockNum)
	}
	commit, err := message.DecodeConsensusMsg(val)
	if err != nil {
		return nil, err
	}
	if err = commit.(*message.Commit).ValidateBasic(); err != nil {
		cp.sabft.log.Error(err)
		return nil, err
	}
	return commit.(*message.Commit), nil
}

func (cp *BFTCheckPoint) ReachCheckPoint(block common.ISignedBlock) (*message.Commit, bool) {
	if ret, ok := cp.cache[block.Id().BlockNum()]; ok {
		cp.sabft.log.Debug("checkpoint reached at height ", block.Id().BlockNum())
		return ret, true
	}
	return nil, false
}

func (cp *BFTCheckPoint) Validate(commit *message.Commit) bool {
	// check +2/3
	if len(cp.sabft.validators)*2/3 >= len(commit.Precommits) {
		cp.sabft.log.Error("checkpoint validate failed, not enough signatures")
		return false
	}

	if !cp.sabft.VerifyCommitSig(commit) {
		cp.sabft.log.Error("checkpoint validate failed, invalid signatures")
		return false
	}

	nextBlockID := common.BlockID{
		Data: commit.ProposedData,
	}
	cp.sabft.log.Infof("checkpoint at block height %v validated.", nextBlockID.BlockNum())
	return true
}
