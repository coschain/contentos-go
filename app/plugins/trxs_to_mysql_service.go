package plugins

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/iservices/service-configs"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/prototype"
	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
	"time"
)


type Op map[string]interface{}

func PurgeOperation(operations []*prototype.Operation) []Op {
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

var TrxMysqlServiceName = "trxmysql"

type TrxMysqlService struct {
	node.Service
	config *service_configs.DatabaseConfig
	inDb  iservices.IDatabaseService
	outDb *sql.DB
	log *logrus.Logger
	ev  EventBus.Bus
	ctx *node.ServiceContext
	quit chan bool
}

func NewTrxMysqlSerVice(ctx *node.ServiceContext, config *service_configs.DatabaseConfig, log *logrus.Logger) (*TrxMysqlService, error) {
	return &TrxMysqlService{ctx: ctx, log: log, config: config}, nil
}

func (t *TrxMysqlService) Start(node *node.Node) error {
	t.quit = make(chan bool)
	inDb, err := t.ctx.Service(iservices.DbServerName)
	if err != nil {
		return err
	}
	t.inDb = inDb.(iservices.IDatabaseService)
	// dns: data source name
	dsn := fmt.Sprintf("%s:%s@/%s", t.config.User, t.config.Password, t.config.Db)
	outDb, err := sql.Open(t.config.Driver, dsn)
	if err != nil {
		return err
	}
	t.outDb = outDb

	ticker := time.NewTimer(time.Second)
	go func() {
		for {
			select {
			case <- ticker.C:
				err := t.pollLIB()
				t.log.Error(err)
			case <- t.quit:
				t.stop()
				break
			}
		}
	}()
	return nil
}

func (t *TrxMysqlService) pollLIB() error {
	var id int32 = 1
	gWrap := table.NewSoGlobalWrap(t.inDb, &id)
	if !gWrap.CheckExist() {
		return errors.New("global wrapper is not exist")
	}
	props := gWrap.GetProps()
	lib := props.IrreversibleBlockNum
	stmt, _ := t.outDb.Prepare("SELECT block_height from libinfo limit 1")
	defer func() {
		if err := stmt.Close(); err != nil {
			t.log.Error(err)
		}
	}()
	var lastLib uint64
	err := stmt.QueryRow().Scan(&lastLib)
	if err != nil {
		if err != sql.ErrNoRows	{
			t.log.Error(err)
		}
	} else {
		// be carefully, no where condition there !!
		// the reason is only one row in the table
		// if introduce the mechanism that record checkpoint, the where closure should be added
		updateStmt, _ := t.outDb.Prepare("UPDATE libinfo SET block_height=?")
		defer func() {
			if err := updateStmt.Close(); err != nil {
				t.log.Error(err)
			}
		}()
		var waitingSyncLib []uint64
		for lastLib <= lib {
			waitingSyncLib = append(waitingSyncLib, lastLib)
			lastLib ++
		}
		for _, lib := range waitingSyncLib {
			t.handleLibNotification(lib)
			_, _ = updateStmt.Exec(lib)
		}
	}
	return nil
}

func (t *TrxMysqlService) handleLibNotification(lib uint64) {
	sWrap := table.NewExtTrxBlockHeightWrap(t.inDb)
	start := lib
	end := lib + 1
	stmt, _ := t.outDb.Prepare("INSERT IGNORE INTO trxinfo (trx_id, block_height, block_id, block_time, invoice, operations)  value (?, ?, ?, ?, ?, ?)")
	_ = sWrap.ForEachByOrder(&start, &end, nil, nil, func(trxKey *prototype.Sha256, blockHeight *uint64, idx uint32) bool {
		if trxKey != nil {
			wrap := table.NewSoExtTrxWrap(t.inDb, trxKey)
			if wrap != nil && wrap.CheckExist() {
				trxId := hex.EncodeToString(trxKey.GetHash())
				blockHeight := wrap.GetBlockHeight()
				blockId := hex.EncodeToString(wrap.GetBlockId().GetHash())
				blockTime := wrap.GetBlockTime().GetUtcSeconds()
				trxWrap := wrap.GetTrxWrap()
				invoice, _ := json.Marshal(trxWrap.Invoice)
				operations := PurgeOperation(trxWrap.SigTrx.GetTrx().GetOperations())
				operationsJson, _ := json.Marshal(operations)
				_, _ = stmt.Exec(trxId, blockHeight, blockId, blockTime, invoice, operationsJson)
				return true
			} else {
				return false
			}
		} else {
			return false
		}
	})
	defer stmt.Close()
}

func (t *TrxMysqlService) stop() {
	_ = t.outDb.Close()
	close(t.quit)
}

func (t *TrxMysqlService) Stop() error {
	//t.unhookEvent()
	t.quit <- true
	return nil
}
