package tests

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/db/storage"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"testing"
	_ "github.com/go-sql-driver/mysql"
)

type Op map[string]interface{}

func ReplaceOperation(operations []*prototype.Operation) []Op {
	var ops []Op
	for _, operation := range operations {
		switch x := operation.Op.(type) {
		case *prototype.Operation_Op1:
			ops = append(ops, Op{"create_account":x.Op1})
		case *prototype.Operation_Op2:
			ops = append(ops, Op{"transfer": x.Op2})
		case *prototype.Operation_Op3:
			ops = append(ops, Op{"bp_register": x.Op3})
		case *prototype.Operation_Op4:
			ops = append(ops, Op{"bp_unregister": x.Op4})
		case *prototype.Operation_Op5:
			ops = append(ops, Op{"bp_vote": x.Op5})
		case *prototype.Operation_Op6:
			ops = append(ops, Op{"post": x.Op6})
		case *prototype.Operation_Op7:
			ops = append(ops, Op{"reply": x.Op7})
		case *prototype.Operation_Op8:
			ops = append(ops, Op{"follow": x.Op8})
		case *prototype.Operation_Op9:
			ops = append(ops, Op{"vote": x.Op9})
		case *prototype.Operation_Op10:
			ops = append(ops, Op{"transfer_to_vesting": x.Op10})
		case *prototype.Operation_Op13:
			ops = append(ops, Op{"contract_deploy": x.Op13})
		case *prototype.Operation_Op14:
			ops = append(ops, Op{"contract_apply": x.Op14})
		case *prototype.Operation_Op15:
			ops = append(ops, Op{"report": x.Op15})
		case *prototype.Operation_Op16:
			ops = append(ops, Op{"convert_vesting": x.Op16})
		}
	}
	return ops
}

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
	stmt, _ := mdb.Prepare("INSERT IGNORE INTO trxinfo (trx_id, block_height, block_id, block_time, invoice, operations)  value (?, ?, ?, ?, ?, ?)")
	var start uint64 = 0
	var end uint64 = 1000
	sWrap := table.NewExtTrxBlockHeightWrap(db)
	_ = sWrap.ForEachByOrder(&start, &end, nil, nil, func(trxKey *prototype.Sha256, blockHeight *uint64, idx uint32) bool {
		if trxKey != nil {
			wrap := table.NewSoExtTrxWrap(db, trxKey)
			if wrap != nil && wrap.CheckExist() {
				trxId := hex.EncodeToString(trxKey.GetHash())
				blockHeight := wrap.GetBlockHeight()
				blockId := hex.EncodeToString(wrap.GetBlockId().GetHash())
				blockTime := wrap.GetBlockTime().GetUtcSeconds()
				trxWrap := wrap.GetTrxWrap()
				invoice, _ := json.Marshal(trxWrap.Invoice)
				operations, _ := json.Marshal(trxWrap.SigTrx.GetTrx().GetOperations())
				_, _ = stmt.Exec(trxId, blockHeight, blockId, blockTime, invoice, operations)
				r, _ := json.Marshal(ReplaceOperation(trxWrap.SigTrx.GetTrx().GetOperations()))
				fmt.Println(string(r))
				return true
			} else {
				return false
			}
		} else {
			return false
		}
	})
	db.Close()
}
