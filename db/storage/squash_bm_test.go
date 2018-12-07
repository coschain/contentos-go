package storage

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
)

const (
	bmBlocks = 1000					// # of blocks to push
	bmTrxsPerBlock = 100			// # of transactions per block
	bmOpsPerTrx = 5					// # of PUT operations per transaction
	bmDelaySquash = 15				// irreversible delay, in blocks
	bmIterRangeSize = 30			// length of range for iteration benchmark

	bmTotalOps = bmOpsPerTrx * bmTrxsPerBlock * bmBlocks
)

//
// Benchmark main entry
//
func Benchmark(b *testing.B) {
	// data initialization
	data := prepareBenchmarkData()
	sortedData := make(sort.StringSlice, len(data))
	copy(sortedData, data)
	sortedData.Sort()

	b.ResetTimer()

	// benchmark squashable
	b.Run("Squashable", makeBenchmarkSuite(false, &data, &sortedData))

	// benchmark revertible + transactional
	b.Run("Revertible", makeBenchmarkSuite(true, &data, &sortedData))
}

// benchmark suite maker
func makeBenchmarkSuite(revertible bool, dataPtr *sort.StringSlice, sortedDataPtr *sort.StringSlice) func(*testing.B) {
	return func(b *testing.B) {
		dbs := openTrxDatabase(b, revertible)
		defer closeTrxDatabase(dbs)

		data := *dataPtr
		sortedData := *sortedDataPtr

		b.Run("pushBlocks", makePushBlocksBenchmark(dbs, &data, revertible))
		b.Run("iteration" + strconv.Itoa(bmIterRangeSize), makeIterationBenchmark(dbs, &sortedData, bmIterRangeSize, false))
		b.Run("reverse-iteration" + strconv.Itoa(bmIterRangeSize), makeIterationBenchmark(dbs, &sortedData, bmIterRangeSize, true))
		b.Run("query", makeQueryBenchmark(dbs, &data))
		b.Run("update", makeUpdateBenchmark(dbs, &data))
		b.Run("delete", makeDeleteBenchmark(dbs, &data))
	}
}

// benchmark maker for block pushing
func makePushBlocksBenchmark(dbs *dbService, dataPtr *sort.StringSlice, revertible bool) func(*testing.B) {
	return func(b *testing.B) {
		idx := 0
		data := *dataPtr
		for i := 0; i < bmBlocks; i++ {
			if revertible {
				dbs.db.BeginTransaction()
			} else {
				dbs.db.BeginTransactionWithTag("block" + strconv.Itoa(i))
			}

			for j := 0; j < bmTrxsPerBlock; j++ {
				dbs.db.BeginTransaction()
				for k := 0; k < bmOpsPerTrx; k++ {
					s := []byte(data[idx])
					dbs.db.Put(s, s)
					idx++
				}
				dbs.db.EndTransaction(true)
			}

			if revertible {
				dbs.db.EndTransaction(true)
			} else if i >= bmDelaySquash {
				dbs.db.Squash("block" + strconv.Itoa(i - bmDelaySquash))
			}
		}
	}
}

// benchmark maker for iteration
func makeIterationBenchmark(dbs *dbService, dataPtr *sort.StringSlice, rangeSize int, reverse bool) func(*testing.B) {
	return func(b *testing.B) {
		data := *dataPtr
		s := len(data)
		for i := 0; i < b.N; i++ {
			start := rand.Intn(s - rangeSize)
			limit := start + rangeSize
			var it Iterator
			if reverse {
				it = dbs.db.NewReversedIterator([]byte(data[start]), []byte(data[limit]))
			} else {
				it = dbs.db.NewIterator([]byte(data[start]), []byte(data[limit]))
			}
			count := 0
			for it.Next() {
				count++
			}
			//b.Log(count)
			dbs.db.DeleteIterator(it)
		}
	}
}

// benchmark maker for querying
func makeQueryBenchmark(dbs *dbService, dataPtr *sort.StringSlice) func(*testing.B) {
	return func(b *testing.B) {
		data := *dataPtr
		s := len(data)
		for i := 0; i < b.N; i++ {
			if _, err := dbs.db.Get([]byte(data[rand.Intn(s)])); err != nil {
				b.Fatal("failed db.get")
			}
		}
	}
}

// benchmark maker for updating
func makeUpdateBenchmark(dbs *dbService, dataPtr *sort.StringSlice) func(*testing.B) {
	return func(b *testing.B) {
		data := *dataPtr
		s := len(data)
		val := []byte(randomString(32))
		for i := 0; i < b.N; i++ {
			dbs.db.Put([]byte(data[rand.Intn(s)]), val)
		}
	}
}

// benchmark maker for deletion
func makeDeleteBenchmark(dbs *dbService, dataPtr *sort.StringSlice) func(*testing.B) {
	return func(b *testing.B) {
		data := *dataPtr
		s := len(data)
		for i := 0; i < b.N; i++ {
			dbs.db.Delete([]byte(data[rand.Intn(s)]))
		}
	}
}


//////////////////////////


// either a squashable or a revertible + transactional
type dbService struct {
	db SquashDatabase
	lvldb *LevelDatabase
	dir string
}

// create a db service
func openTrxDatabase(b *testing.B, revertible bool) *dbService {
	dir, err := ioutil.TempDir("", "squash_bm_test_lvldb")
	if err != nil {
		b.Fatal(err)
	}
	fn := filepath.Join(dir, randomString(8))
	db, err := NewLevelDatabase(fn)
	if err != nil {
		b.Fatal(err)
	}
	var sdb SquashDatabase
	if revertible {
		sdb = NewSquashableDatabase(NewRevertibleDatabase(db), true)
	} else {
		sdb = NewSquashableDatabase(db, true)
	}
	return &dbService{
		db: sdb,
		lvldb: db,
		dir: dir,
	}
}

// close a db service
func closeTrxDatabase(dbsvc *dbService) {
	dbsvc.lvldb.Close()
	os.RemoveAll(dbsvc.dir)
}

// data generation
func prepareBenchmarkData() sort.StringSlice {
	data := make(sort.StringSlice, bmTotalOps)
	for i := 0; i < bmTotalOps; i++ {
		data[i] = fmt.Sprintf("%x", md5.Sum([]byte(strconv.Itoa(i))))
	}
	return data
}
