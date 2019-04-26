package app

import (
	"fmt"
	"github.com/coschain/contentos-go/db/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestBlockIceberg(t *testing.T) {
	var n uint64
	a := assert.New(t)

	dir, err := ioutil.TempDir("", "block_iceberg")
	a.NoError(err, "temp directory creation failed")
	defer os.RemoveAll(dir)

	fn := filepath.Join(dir, strconv.FormatUint(rand.Uint64(), 10))
	db, _ := storage.NewDatabase(fn)
	a.NotNil(db, "database service creation failed")
	a.NoError(db.Start(nil), "database service start failed")
	defer db.Stop()

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// create instance based on an empty db
	berg := NewBlockIceberg(db, logger, false)
	a.NotNil(berg, "iceberg creation failed")

	// only BeginBlock(1) is allowed for an empty db. everything else must returns error.
	n, _ = berg.LastFinalizedBlock()
	a.Equal(uint64(0), n, "finalized block on empty db should be 0 (meaning no finalized block yet)")
	n, _, err = berg.LatestBlock()
	a.Equal(uint64(0), n, "latest block on empty db should be 0 (meaning no block yet)")
	a.Error(berg.BeginBlock(0), "invalid block number 0")
	a.Error(berg.BeginBlock(2), "block number 2 is illegal without block #1")
	a.Error(berg.EndBlock(false), "invalid EndBlock(false)")
	a.Error(berg.EndBlock(true), "invalid EndBlock(true)")
	a.Error(berg.RevertBlock(0), "invalid RevertBlock(0)")
	a.Error(berg.RevertBlock(1), "invalid RevertBlock(1)")
	a.Error(berg.RevertBlock(2), "invalid RevertBlock(2)")
	a.Error(berg.FinalizeBlock(0), "invalid FinalizeBlock(0)")
	a.Error(berg.FinalizeBlock(1), "invalid FinalizeBlock(1)")
	a.Error(berg.FinalizeBlock(2), "invalid FinalizeBlock(2)")

	// BeginBlock(1) -> EndBlock(false)
	a.NoError(berg.BeginBlock(1))
	a.NoError(db.Put([]byte("k1"), []byte("v1")))
	a.NoError(berg.EndBlock(false))
	_, err = db.Get([]byte("k1"))
	a.Error(err, "EndBlock(false) should discard (k1, v1)")

	// BeginBlock(1) -> EndBlock(true)
	a.NoError(berg.BeginBlock(1))
	a.NoError(db.Put([]byte("k1"), []byte("v1")))
	a.NoError(berg.EndBlock(true))
	_, err = db.Get([]byte("k1"))
	a.NoError(err, "EndBlock(true) should commit (k1, v1)")

	// unexpected EndBlock()
	a.Error(berg.EndBlock(false), "invalid EndBlock(false)")
	a.Error(berg.EndBlock(true), "invalid EndBlock(true)")

	// non-consecutive block number
	a.Error(berg.BeginBlock(3))
	a.Error(berg.EndBlock(false), "invalid EndBlock(false)")

	// revert block 1
	a.NoError(berg.RevertBlock(1))
	_, err = db.Get([]byte("k1"))
	a.Error(err, "RevertBlock(1) should remove (k1, v1)")

	// block 1 & 2
	a.NoError(berg.BeginBlock(1))
	a.NoError(db.Put([]byte("k1"), []byte("v1")))
	a.NoError(berg.EndBlock(true))
	a.NoError(berg.BeginBlock(2))
	a.NoError(db.Put([]byte("k2"), []byte("v2")))
	a.NoError(berg.EndBlock(true))
	_, err = db.Get([]byte("k1"))
	a.NoError(err, "EndBlock(true) for block 1 should commit (k1, v1)")
	_, err = db.Get([]byte("k2"))
	a.NoError(err, "EndBlock(true) for block 2 should commit (k2, v2)")

	// finalize block 1
	a.NoError(berg.FinalizeBlock(1))

	// revert a finalized block
	a.Error(berg.RevertBlock(1), "block 1 finalized, thus can't be reverted")

	// non-finalized blocks can be reverted
	a.NoError(berg.RevertBlock(2), "block 2 should be reverted")
	_, err = db.Get([]byte("k2"))
	a.Error(err, "RevertBlock(2) should remove (k2, v2)")

	// blocks 2...1000
	for i := 2; i <= 1000; i++ {
		a.NoError(berg.BeginBlock(uint64(i)))
		a.NoError(db.Put([]byte(fmt.Sprintf("k%d", i)), []byte(fmt.Sprintf("k%d", i))))
		a.NoError(berg.EndBlock(true))
		a.True(db.TransactionHeight() <= defaultBlockIcebergHighWM, "max in-memory blocks")
	}
	for i := 1; i <= 1000; i++ {
		_, err = db.Get([]byte(fmt.Sprintf("k%d", i)))
		a.NoErrorf(err, "k%d should exist", i)
	}

	// revert to some in-memory block
	a.NoError(berg.RevertBlock(990))
	n, err = berg.LastFinalizedBlock()
	a.Equal(uint64(1), n, "last finalized should be 1")
	n, _, err = berg.LatestBlock()
	a.Equal(uint64(989), n, "last block should be 989")
	for i := 990; i <= 1000; i++ {
		_, err = db.Get([]byte(fmt.Sprintf("k%d", i)))
		a.Errorf(err, "k%d should be removed by RevertBlock(990)", i)
	}
	for i := 1; i <= 989; i++ {
		_, err = db.Get([]byte(fmt.Sprintf("k%d", i)))
		a.NoErrorf(err, "k%d should survive RevertBlock(990)", i)
	}

	// finalize some in-db block
	n = uint64(db.TransactionHeight())
	a.NoError(berg.FinalizeBlock(100))
	a.Equal(n, uint64(db.TransactionHeight()))
	n, err = berg.LastFinalizedBlock()
	a.Equal(uint64(100), n)
	n, _, err = berg.LatestBlock()
	a.Equal(uint64(989), n)

	// revert finalized blocks
	a.Error(berg.RevertBlock(100))
	a.Error(berg.RevertBlock(99))
	a.Error(berg.RevertBlock(3))

	// finalize in-memory block
	a.NoError(berg.FinalizeBlock(985))
	a.Equal(uint64(4), uint64(db.TransactionHeight()))
	n, err = berg.LastFinalizedBlock()
	a.Equal(uint64(985), n)
	n, _, err = berg.LatestBlock()
	a.Equal(uint64(989), n)
	a.Error(berg.RevertBlock(985))
	a.Error(berg.RevertBlock(800))
	a.Error(berg.RevertBlock(20))
	for i := 1; i <= 989; i++ {
		_, err = db.Get([]byte(fmt.Sprintf("k%d", i)))
		a.NoErrorf(err, "k%d should not be touched during block finalizations", i)
	}

	a.NoError(berg.RevertBlock(986))
	n, err = berg.LastFinalizedBlock()
	a.Equal(uint64(985), n)
	n, _, err = berg.LatestBlock()
	a.Equal(uint64(985), n)
	a.Equal(uint64(0), uint64(db.TransactionHeight()))
	_, err = db.Get([]byte("k986"))
	a.Errorf(err, "k986 should be removed by RevertBlock(986)")
	_, err = db.Get([]byte("k985"))
	a.NoErrorf(err, "k985 should survive RevertBlock(986)")

	// re-create block iceberg
	a.NoError(db.Stop())
	a.NoError(db.Start(nil))
	berg = NewBlockIceberg(db, logger, false)
	a.NotNil(berg)

	// check latest & finalized block number
	n, err = berg.LastFinalizedBlock()
	a.Equal(uint64(985), n)
	n, _, err = berg.LatestBlock()
	a.Equal(uint64(985), n)

	// add more blocks
	for i := 986; i <= 1200; i++ {
		a.NoError(berg.BeginBlock(uint64(i)))
		a.NoError(db.Put([]byte(fmt.Sprintf("k%d", i)), []byte(fmt.Sprintf("k%d", i))))
		a.NoError(berg.EndBlock(true))
		a.True(db.TransactionHeight() <= defaultBlockIcebergHighWM, "max in-memory blocks")
	}

	// revert a in-db block
	berg.RevertBlock(1000)
	for i := 1000; i <= 1200; i++ {
		_, err = db.Get([]byte(fmt.Sprintf("k%d", i)))
		a.Errorf(err, "k%d should be removed by RevertBlock(1000)", i)
	}
	for i := 1; i <= 999; i++ {
		_, err = db.Get([]byte(fmt.Sprintf("k%d", i)))
		a.NoErrorf(err, "k%d should survive RevertBlock(1000)", i)
	}
}
