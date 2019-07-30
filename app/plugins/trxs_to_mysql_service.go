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
		ops = append(ops, Op{prototype.GetGenericOperationName(operation): prototype.GetBaseOperation(operation)})
	}
	return ops
}

func FindCreator(operation *prototype.Operation) (name string) {
	signers := make(map[string]bool)
	prototype.GetBaseOperation(operation).GetSigner(&signers)
	if len(signers) > 0 {
		for s := range signers {
			name = s
			break
		}
	}
	return
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

var TrxMysqlServiceName = "trxsqlservice"

type TrxMysqlService struct {
	node.Service
	config *service_configs.DatabaseConfig
	consensus iservices.IConsensus
	outDb *sql.DB
	log *logrus.Logger
	ctx *node.ServiceContext
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
		if count > 1000 {
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
	blk := blks[0].(*prototype.SignedBlock)
	for _, trx := range blk.Transactions {
		trxHash, _ := trx.SigTrx.GetTrxHash(t.ctx.ChainId())
		trxId := hex.EncodeToString(trxHash)
		blockHeight := lib
		data := blk.Id().Data
		blockId := hex.EncodeToString(data[:])
		blockTime := blk.Timestamp()
		invoice, _ := json.Marshal(trx.Receipt)
		operations := PurgeOperation(trx.SigTrx.GetTrx().GetOperations())
		operationsJson, _ := json.Marshal(operations)
		//operation := trx.SigTrx.GetTrx().GetOperations()[0]
		creator := FindCreator(trx.SigTrx.GetTrx().GetOperations()[0])
		_, _ = t.outDb.Exec("INSERT IGNORE INTO trxinfo (trx_id, block_height, block_id, block_time, invoice, operations, creator)  value (?, ?, ?, ?, ?, ?, ?)", trxId, blockHeight, blockId, blockTime, invoice, operationsJson, creator)
		for _, operation := range trx.SigTrx.GetTrx().GetOperations() {
			if IsCreateAccountOp(operation) {
				_, _ = t.outDb.Exec("INSERT IGNORE INTO createaccountinfo (trx_id, create_time, creator, pubkey, account) values (?, ?, ?, ?, ?)", trxId, blockTime, creator, operation.GetOp1().PubKey.ToWIF(), operation.GetOp1().NewAccountName.Value)
				break
			}
		}
		for _, operation := range trx.SigTrx.GetTrx().GetOperations() {
			if IsTransferOp(operation) {
				_, _ = t.outDb.Exec("INSERT IGNORE INTO transferinfo (trx_id, create_time, sender, receiver, amount, memo) values (?, ?, ?, ?, ?, ?)", trxId, blockTime, creator, operation.GetOp2().To.Value, operation.GetOp2().Amount.Value, operation.GetOp2().Memo)
				break
			}
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