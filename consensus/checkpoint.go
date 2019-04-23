package consensus

import (
	"encoding/binary"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/db/storage"
	"github.com/coschain/gobft/message"
)

/***************************
validators:
	if reach consensus:
		add checkPoint
		if do commit success:
			flush checkPoint
		else if committing a missing block:
			do sync

	recv message.Commit:
		if not committed already:
			pass message.Commit to gobft

non-validators:
	recv message.Commit:
		if not committed already && num within range:
			add checkPoint
			if lib num == (message.Commit).LastCommitted:
				if validate(message.Commit)==true:
					if has the block about to be committed:
						if do commit success:
							flush checkPoint
							commit later blocks if possible
					else:
						do sync
				else:
					remove checkPoint

push block b:
	if b.id == next_checkPoint.id:
		if do commit success:
			flush checkPoint
***************************/

type BFTCheckPoint struct {
	sabft   *SABFT
	dataDir string
	db      storage.Database

	lastCommitted common.BlockID
	nextCP        common.BlockID
	cache         map[common.BlockID]*message.Commit // lastCommitted-->Commit
	//futurnCPs *list.List
}

func NewBFTCheckPoint(dir string, sabft *SABFT) *BFTCheckPoint {
	db, err := storage.NewLevelDatabase(dir)
	if err != nil {
		panic(err)
	}
	return &BFTCheckPoint{
		sabft:         sabft,
		dataDir:       dir,
		db:            db,
		lastCommitted: common.EmptyBlockID,
		nextCP:        common.EmptyBlockID,
		cache:         make(map[common.BlockID]*message.Commit),
	}
}

func (cp *BFTCheckPoint) Flush() error {
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, cp.nextCP.BlockNum())
	err := cp.db.Put(key, cp.cache[cp.lastCommitted].Bytes())
	if err != nil {
		cp.sabft.log.Fatal(err)
		return err
	}
	delete(cp.cache, cp.lastCommitted)

	cp.lastCommitted = cp.nextCP
	cp.sabft.log.Info("checkpoint flushed at block height ", cp.lastCommitted)
	cp.nextCP = common.EmptyBlockID
	if v, ok := cp.cache[cp.lastCommitted]; ok {
		cp.nextCP = ConvertToBlockID(v.ProposedData)
	}
	return nil
}

func (cp *BFTCheckPoint) Add(commit *message.Commit) bool {
	if err := commit.ValidateBasic(); err != nil {
		cp.sabft.log.Error(err)
		return false
	}
	blockID := ExtractBlockID(commit)
	blockNum := blockID.BlockNum()
	libNum := cp.lastCommitted.BlockNum()
	if blockNum > libNum+constants.MaxUncommittedBlockNum ||
		blockNum <= libNum {
		return false
	}

	prev := ConvertToBlockID(commit.Prev)
	if _, ok := cp.cache[prev]; ok {
		return false
	}
	cp.cache[prev] = commit
	if cp.lastCommitted == common.EmptyBlockID && cp.lastCommitted == prev {
		cp.nextCP = blockID
	}
	cp.sabft.log.Info("CheckPoint added", commit.ProposedData)
	return true
}

func (cp *BFTCheckPoint) NextUncommitted() *message.Commit {
	if v, ok := cp.cache[cp.lastCommitted]; ok {
		return v
	}
	return nil
}

func (cp *BFTCheckPoint) RemoveNextUncommitted() {
	delete(cp.cache, cp.lastCommitted)
	cp.nextCP = common.EmptyBlockID
}

func (cp *BFTCheckPoint) ReachCheckPoint(commit *message.Commit) bool {
	id := ExtractBlockID(commit)
	if id == common.EmptyBlockID {
		cp.sabft.log.Fatal("checkpoint on an empty block")
		return false
	}
	_, ok := cp.cache[cp.lastCommitted]
	if !ok {
		return false
	}
	return cp.nextCP == id // && ConvertToBlockID(v.Prev) == cp.lastCommitted
}

func (cp *BFTCheckPoint) Validate(commit *message.Commit) bool {
	// TODO: base validators on last committed block
	//if !cp.sabft.VerifyCommitSig(commit) {
	//	return false
	//}
	return true
}

func (cp *BFTCheckPoint) GetNext(blockNum uint64) (*message.Commit, error) {
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, blockNum+1)
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
	if err = commit.(*message.Commit).ValidateBasic(); err != nil {
		cp.sabft.log.Error(err)
		return nil, err
	}
	return commit.(*message.Commit), nil
}
