package plugins

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

func FindCreator(operation *prototype.Operation) string {
	switch x := operation.Op.(type) {
	case *prototype.Operation_Op1:
		return x.Op1.Creator.Value
	case *prototype.Operation_Op2:
		return x.Op2.From.Value
	case *prototype.Operation_Op3:
		return x.Op3.Owner.Value
	case *prototype.Operation_Op4:
		return x.Op4.Owner.Value
	case *prototype.Operation_Op5:
		return x.Op5.Voter.Value
	case *prototype.Operation_Op6:
		return x.Op6.Owner.Value
	case *prototype.Operation_Op7:
		return x.Op7.Owner.Value
	case *prototype.Operation_Op8:
		return x.Op8.Account.Value
	case *prototype.Operation_Op9:
		return x.Op9.Voter.Value
	case *prototype.Operation_Op10:
		return x.Op10.From.Value
	case *prototype.Operation_Op13:
		return x.Op13.Owner.Value
	case *prototype.Operation_Op14:
		return x.Op14.Caller.Value
	case *prototype.Operation_Op15:
		return x.Op15.Reporter.Value
	case *prototype.Operation_Op16:
		return x.Op16.From.Value
	}
	return ""
}

func IsCreateAccountOp(operation *prototype.Operation) bool {
	switch operation.Op.(type) {
	case *prototype.Operation_Op1:
		return true
	default:
		return false
	}
}

func IsTransferOp(operation *prototype.Operation) bool {
	switch operation.Op.(type) {
	case *prototype.Operation_Op2:
		return true
	default:
		return false
	}
}

var TrxMysqlServiceName = "trxmysql"

type TrxMysqlService struct {
	node.Service
	config *service_configs.DatabaseConfig
	consensus iservices.IConsensus
	outDb *sql.DB
	log *logrus.Logger
	ctx *node.ServiceContext
	//timer *time.Timer
	//ticker *time.Ticker
	quit chan bool
}

func NewTrxMysqlSerVice(ctx *node.ServiceContext, config *service_configs.DatabaseConfig, log *logrus.Logger) (*TrxMysqlService, error) {
	return &TrxMysqlService{ctx: ctx, log: log, config: config}, nil
}

func (t *TrxMysqlService) Start(node *node.Node) error {
	t.quit = make(chan bool)
	consensus, err := t.ctx.Service(iservices.ConsensusServerName)
	if err != nil {
		return err
	}
	t.consensus = consensus.(iservices.IConsensus)
	// dns: data source name
	dsn := fmt.Sprintf("%s:%s@/%s", t.config.User, t.config.Password, t.config.Db)
	outDb, err := sql.Open(t.config.Driver, dsn)

	if err != nil {
		return err
	}
	t.outDb = outDb

	ticker := time.NewTicker(time.Second)
	go func() {
		for {
			select {
			case <- ticker.C:
				if err := t.pollLIB(); err != nil {
					t.log.Error(err)
				}
			case <- t.quit:
				ticker.Stop()
				t.stop()
				return
			}
		}
	}()
	return nil
}

func (t *TrxMysqlService) pollLIB() error {
	start := time.Now()
	lib := t.consensus.GetLIB().BlockNum()
	t.log.Debugf("[trx db] sync lib: %d \n", lib)
	stmt, _ := t.outDb.Prepare("SELECT lib from libinfo limit 1")
	defer stmt.Close()
	var lastLib uint64 = 0
	_ = stmt.QueryRow().Scan(&lastLib)
	// be carefully, no where condition there !!
	// the reason is only one row in the table
	// if introduce the mechanism that record checkpoint, the where closure should be added
	updateStmt, _ := t.outDb.Prepare("UPDATE libinfo SET lib=?, last_check_time=?")
	defer updateStmt.Close()
	var waitingSyncLib []uint64
	var count = 0
	for lastLib < lib {
		if count > 100 {
			break
		}
		waitingSyncLib = append(waitingSyncLib, lastLib)
		lastLib ++
		count ++
	}

	for _, block := range waitingSyncLib {
		blockStart := time.Now()
		t.handleLibNotification(block)
		utcTimestamp := time.Now().UTC().Unix()
		_, _ = updateStmt.Exec(block, utcTimestamp)
		t.log.Debugf("[trx db] insert block %d, spent: %v", block, time.Now().Sub(blockStart))
	}
	t.log.Debugf("[trx db] PollLib spent: %v", time.Now().Sub(start))
	return nil
}

func (t *TrxMysqlService) handleLibNotification(lib uint64) {
	blks , err := t.consensus.FetchBlocks(lib, lib)
	if err != nil {
		t.log.Error(err)
		return
	}
	if len(blks) == 0 {
		return
	}
	stmt, _ := t.outDb.Prepare("INSERT IGNORE INTO trxinfo (trx_id, block_height, block_id, block_time, invoice, operations, creator)  value (?, ?, ?, ?, ?, ?, ?)")
	defer stmt.Close()
	accountStmt, _ := t.outDb.Prepare("INSERT IGNORE INTO createaccountinfo (trx_id, create_time, creator, pubkey, account) values (?, ?, ?, ?, ?)")
	defer accountStmt.Close()
	transferStmt, _ := t.outDb.Prepare("INSERT IGNORE INTO transferinfo (trx_id, create_time, sender, receiver, amount, memo) values (?, ?, ?, ?, ?, ?)")
	defer transferStmt.Close()
	blk := blks[0].(*prototype.SignedBlock)
	for _, trx := range blk.Transactions {
		cid := prototype.ChainId{Value: 0}
		trxHash, _ := trx.SigTrx.GetTrxHash(cid)
		trxId := hex.EncodeToString(trxHash)
		blockHeight := lib
		data := blk.Id().Data
		blockId := hex.EncodeToString(data[:])
		blockTime := blk.Timestamp()
		invoice, _ := json.Marshal(trx.Invoice)
		operations := PurgeOperation(trx.SigTrx.GetTrx().GetOperations())
		operationsJson, _ := json.Marshal(operations)
		operation := trx.SigTrx.GetTrx().GetOperations()[0]
		creator := FindCreator(operation)
		_, _ = stmt.Exec(trxId, blockHeight, blockId, blockTime, invoice, operationsJson, creator)
		if IsCreateAccountOp(operation) {
			_, _ = accountStmt.Exec(trxId, blockTime, creator, operation.GetOp1().Owner.ToWIF(),  operation.GetOp1().NewAccountName.Value)
		}
		if IsTransferOp(operation) {
			_, _ = transferStmt.Exec(trxId, blockTime, creator, operation.GetOp2().To.Value, operation.GetOp2().Amount.Value, operation.GetOp2().Memo)
		}
	}
}

func (t *TrxMysqlService) stop() {
	_ = t.outDb.Close()
	//t.ticker.Stop()
}

func (t *TrxMysqlService) Stop() error {
	//t.unhookEvent()
	t.quit <- true
	close(t.quit)
	return nil
}
