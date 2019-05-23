package tests

import (
	"database/sql"
	"fmt"
	"github.com/coschain/contentos-go/db/storage"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	_ "github.com/go-sql-driver/mysql"
	"testing"
)

type Op map[string]interface{}

func ReplaceOperation(operations []*prototype.Operation) []Op {
	var ops []Op
	for _, operation := range operations {
		ops = append(ops, Op{prototype.GetGenericOperationName(operation): prototype.GetBaseOperation(operation)})
	}
	return ops
}

//func pollLIB() error {
//	var id int32 = 1
//	gWrap := table.NewSoGlobalWrap(t.inDb, &id)
//	if !gWrap.CheckExist() {
//		return errors.New("global wrapper is not exist")
//	}
//	props := gWrap.GetProps()
//	lib := props.IrreversibleBlockNum
//	stmt, _ := t.outDb.Prepare("SELECT lib from libinfo limit 1")
//	defer func() {
//		if err := stmt.Close(); err != nil {
//			t.log.Error(err)
//		}
//	}()
//	var lastLib uint64 = 0
//	err := stmt.QueryRow().Scan(&lastLib)
//	if err != nil {
//		if err != sql.ErrNoRows	{
//			t.log.Error(err)
//		}
//	} else {
//		// be carefully, no where condition there !!
//		// the reason is only one row in the table
//		// if introduce the mechanism that record checkpoint, the where closure should be added
//		updateStmt, _ := t.outDb.Prepare("UPDATE libinfo SET lib=?, last_check_time=?")
//		defer func() {
//			if err := updateStmt.Close(); err != nil {
//				t.log.Error(err)
//			}
//		}()
//		var waitingSyncLib []uint64
//		for lastLib <= lib {
//			waitingSyncLib = append(waitingSyncLib, lastLib)
//			lastLib ++
//		}
//		for _, lib := range waitingSyncLib {
//			t.handleLibNotification(lib)
//			utcTimestamp := time.Now().UTC().Unix()
//			_, _ = updateStmt.Exec(lib, utcTimestamp)
//		}
//	}
//	return nil
//}

func TestTrxInfoInsert(t *testing.T) {
	ds, err := storage.NewDatabase("/Users/aprocysanae/.coschain/testcosd_0/db")
	if err != nil {
		fmt.Println(err)
	}
	dbService := &storage.GuardedDatabaseService{DatabaseService: *ds}
	err = dbService.Start(nil)
	if err != nil {
		fmt.Println(err)
	}
	var db iservices.IDatabaseService
	db = dbService
	mdb, err := sql.Open("mysql", "contentos:123456@/contentosdb")
	if err != nil {
		fmt.Println(err)
	}

	//var id int32 = 1
	//gWrap := table.NewSoGlobalWrap(db, &id)
	//props := gWrap.GetProps()
	//lib := props.IrreversibleBlockNum
	//fmt.Println(lib)
	selectStmt, _ := mdb.Prepare("SELECT lib from libinfo limit 1")
	var lastLib uint64 = 0
	_ = selectStmt.QueryRow().Scan(&lastLib)
	//fmt.Println(err)
	//fmt.Println(lastLib)
	//updateStmt, _ := mdb.Prepare("UPDATE libinfo SET lib=?, last_check_time=?")
	//defer updateStmt.Close()
	//utcTimestamp := time.Now().UTC().Unix()
	//_, _ = updateStmt.Exec(lib, utcTimestamp)
	//var waitingSyncLib []uint64
	//for lastLib <= lib {
	//	waitingSyncLib = append(waitingSyncLib, lastLib)
	//	lastLib ++
	//}
	//stmt, _ := mdb.Prepare("INSERT IGNORE INTO trxinfo (trx_id, block_height, block_id, block_time, invoice, operations)  value (?, ?, ?, ?, ?, ?)")
	//var start uint64 = 0
	//var end uint64 = 1000
	//sWrap := table.NewExtTrxBlockHeightWrap(db)
	//_ = sWrap.ForEachByOrder(&start, &end, nil, nil, func(trxKey *prototype.Sha256, blockHeight *uint64, idx uint32) bool {
	//	if trxKey != nil {
	//		wrap := table.NewSoExtTrxWrap(db, trxKey)
	//		if wrap != nil && wrap.CheckExist() {
	//			trxId := hex.EncodeToString(trxKey.GetHash())
	//			blockHeight := wrap.GetBlockHeight()
	//			blockId := hex.EncodeToString(wrap.GetBlockId().GetHash())
	//			blockTime := wrap.GetBlockTime().GetUtcSeconds()
	//			trxWrap := wrap.GetTrxWrap()
	//			invoice, _ := json.Marshal(trxWrap.Invoice)
	//			operations, _ := json.Marshal(trxWrap.SigTrx.GetTrx().GetOperations())
	//			_, _ = stmt.Exec(trxId, blockHeight, blockId, blockTime, invoice, operations)
	//			r, _ := json.Marshal(ReplaceOperation(trxWrap.SigTrx.GetTrx().GetOperations()))
	//			fmt.Println(string(r))
	//			return true
	//		} else {
	//			return false
	//		}
	//	} else {
	//		return false
	//	}
	//})
	db.Close()
}


