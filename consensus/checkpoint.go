package consensus

import (
	"encoding/binary"
	"fmt"
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

// BFTCheckPoint maintains the bft consensus evidence, the votes collected
// for the same checkpoint in different validators might differ. But all
// nodes including validators should have the same number of checkpoints with
// exact same order.
// all methods have time complexity of O(1)
type BFTCheckPoint struct {
	sabft   *SABFT
	dataDir string
	db      storage.Database

	lastCommitted common.BlockID
	nextCP        common.BlockID
	cache         *CommitCache // lastCommitted-->Commit

	indexPrefix [8]byte
}

func NewBFTCheckPoint(dir string, sabft *SABFT) *BFTCheckPoint {
	db, err := storage.NewLevelDatabase(dir)
	if err != nil {
		panic(err)
	}
	lc := sabft.ForkDB.LastCommitted()
	return &BFTCheckPoint{
		sabft:         sabft,
		dataDir:       dir,
		db:            db,
		lastCommitted: lc,
		nextCP:        common.EmptyBlockID,
		cache:         NewCommitCache(),
		indexPrefix:   [8]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
	}
}

func (cp *BFTCheckPoint) Close() {
	cp.db.Close()
}

func (cp *BFTCheckPoint) getIdxKey(idx uint64) []byte {
	idxKey := make([]byte, 16)
	copy(idxKey, cp.indexPrefix[:])
	binary.BigEndian.PutUint64(idxKey[8:], idx)
	return idxKey
}

func (cp *BFTCheckPoint) Flush(bid common.BlockID) error {
	key := make([]byte, 8)
	for {
		if cp.cache.Get(cp.lastCommitted) == nil {
			errstr := fmt.Sprintf("*********** %s lc %v/ cp lc %v nextCP %v, commit_cache %s",
				cp.sabft.Name, cp.sabft.ForkDB.LastCommitted(), cp.lastCommitted, cp.nextCP, cp.cache.String())
			cp.sabft.log.Error(errstr)
			panic(errstr)
		}

		binary.BigEndian.PutUint64(key, cp.nextCP.BlockNum())
		batch := cp.db.NewBatch()
		err := batch.Put(key, cp.cache.Get(cp.lastCommitted).Bytes())
		if err != nil {
			cp.sabft.log.Fatalf("BFT-Flush: %v",err)
			return err
		}

		err = batch.Put(cp.getIdxKey(uint64(cp.cache.Get(cp.lastCommitted).Height())), key)
		if err != nil {
			cp.sabft.log.Fatalf("BFT-Flush: %v",err)
			return err
		}
		if err = batch.Write(); err != nil {
			panic(fmt.Sprintf("BFT-Flush error: %v", err) )
		}

		//delete(cp.cache, cp.lastCommitted)
		cp.cache.Commit(bid)

		cp.lastCommitted = cp.nextCP
		cp.sabft.log.Info("checkpoint flushed at block height ", cp.nextCP.BlockNum())
		cp.nextCP = common.EmptyBlockID
		if v := cp.cache.Get(cp.lastCommitted); v != nil {
			cp.nextCP = ConvertToBlockID(v.ProposedData)
		}
		if cp.lastCommitted == bid {
			// TODO: purge garbage Commit in cache
			return nil
		}
		if cp.nextCP == common.EmptyBlockID {
			break
		}
	}
	cp.sabft.log.Warnf("checkpoint flushing interrupted after block height %d, meant to flush to %d",
		cp.lastCommitted.BlockNum(), bid.BlockNum())
	return nil
}

func (cp *BFTCheckPoint) Add(commit *message.Commit) error {
	if err := commit.ValidateBasic(); err != nil {
		cp.sabft.log.Error(err)
		return ErrInvalidCheckPoint
	}
	blockID := ExtractBlockID(commit)
	blockNum := blockID.BlockNum()
	libNum := cp.lastCommitted.BlockNum()
	if blockNum > libNum+constants.MaxUncommittedBlockNum ||
		blockNum <= libNum {
		cp.sabft.log.Errorf("last commit: %d, commit num: %d, err: %s",
			libNum, blockNum, ErrCheckPointOutOfRange.Error())
		return ErrCheckPointOutOfRange
	}

	if !cp.cache.Add(commit) {
		return nil
	}
	prev := ConvertToBlockID(commit.Prev)
	if cp.lastCommitted == prev {
		cp.nextCP = blockID
	}
	cp.sabft.log.Infof("CheckPoint added %v %d, prev = %v, lib = %d",
		blockID, blockID.BlockNum(), commit.Prev, cp.sabft.ForkDB.LastCommitted().BlockNum())
	return nil
}

func (cp *BFTCheckPoint) Remove(commit *message.Commit) {
	if cp.lastCommitted != ConvertToBlockID(commit.Prev) {
		panic("removing a invalid checkpoint")
	}
	cp.cache.Remove(cp.lastCommitted)
	cp.nextCP = common.EmptyBlockID
}

func (cp *BFTCheckPoint) HasDanglingCheckPoint() bool {
	return cp.NextUncommitted() == nil && cp.cache.HasDangling()
}

// (from, to)
// @from is the last committed checkpoint
// @to is any of the dangling uncommitted checkpoints
// Only call it if HasDanglingCheckPoint returns true
func (cp *BFTCheckPoint) MissingRange() (from, to uint64) {
	return cp.lastCommitted.BlockNum(), cp.cache.GetDanglingHeight()
}

func (cp *BFTCheckPoint) NextUncommitted() *message.Commit {
	return cp.cache.Get(cp.lastCommitted)
}

func (cp *BFTCheckPoint) RemoveNextUncommitted() {
	cp.cache.Remove(cp.lastCommitted)
	cp.nextCP = common.EmptyBlockID
}

func (cp *BFTCheckPoint) IsNextCheckPoint(commit *message.Commit) bool {
	id := ExtractBlockID(commit)
	if id == common.EmptyBlockID {
		cp.sabft.log.Fatal("checkpoint on an empty block")
		return false
	}
	cp.sabft.log.Warn("cp.nextCP: ", cp.nextCP.BlockNum(), " commit number: ", id.BlockNum())
	//_, ok := cp.cache[cp.lastCommitted]
	//if !ok {
	if cp.cache.Get(cp.lastCommitted) == nil {
		cp.sabft.log.Warn("cp not in cache, cp.lastCommitted: ", cp.lastCommitted.BlockNum(), " commit: ", commit)
		return false
	}
	return cp.nextCP == id // && ConvertToBlockID(v.Prev) == cp.lastCommitted
}

func (cp *BFTCheckPoint) Validate(commit *message.Commit) bool {
	if !cp.sabft.verifyCommitSig(commit) {
		return false
	}
	return true
}

func (cp *BFTCheckPoint) GetNext(blockNum uint64) (*message.Commit, error) {
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, blockNum+1)
	var val []byte
	cp.db.Iterate(key, cp.indexPrefix[:], false, func(key, value []byte) bool {
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

func (cp *BFTCheckPoint) GetIth(i uint64) (*message.Commit, error) {
	idxKey := cp.getIdxKey(i)
	blockNum, err := cp.db.Get(idxKey)
	if err != nil {
		cp.sabft.log.Error(err)
		return nil, err
	}
	c, err := cp.db.Get(blockNum)
	if err != nil {
		cp.sabft.log.Error(err)
		return nil, err
	}
	commit, err := message.DecodeConsensusMsg(c)
	if err != nil {
		return nil, err
	}
	return commit.(*message.Commit), nil
}
