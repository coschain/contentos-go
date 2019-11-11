package plugins

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/iservices/service-configs"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/prototype"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
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

var TrxMysqlServiceName = "trxsqlservice"

type TrxInfo struct {
	ID uint64					`gorm:"primary_key;auto_increment"`
	TrxId string `gorm:"unique_index;not null"`
	BlockHeight uint64			`gorm:"index;not null"`
	BlockTime uint64 `gorm:"index;not null"`
	Invoice json.RawMessage `sql:"type:json"`
	Operations json.RawMessage `sql:"type:json"`
	BlockId string `gorm:"not null"`
	Creator string `gorm:"index;not null"`
}

func (TrxInfo) TableName() string {
	return "trxinfo"
}

type LibInfo struct {
	Lib uint64
	LastCheckTime  int64
}

func (LibInfo) TableName() string {
	return "libinfo"
}

type TrxMysqlService struct {
	node.Service
	config *service_configs.DatabaseConfig
	consensus iservices.IConsensus
	outDb *gorm.DB
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
	//outDb, err := sql.Open(t.config.Driver, dsn)
	if db, err := gorm.Open(t.config.Driver, dsn); err != nil {
		return err
	} else {
		t.outDb = db
	}

	if !t.outDb.HasTable(&TrxInfo{}) {
		if err := t.outDb.CreateTable(&TrxInfo{}).Error; err != nil {
			_ = t.outDb.Close()
			return err
		}
	}
	progress := &LibInfo{
		Lib: 0,
		LastCheckTime:  0,
	}
	if !t.outDb.HasTable(progress) {
		if err := t.outDb.CreateTable(progress).Error; err != nil {
			_ = t.outDb.Close()
			return err
		}
		t.outDb.Create(progress)
	}

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
	start := common.EasyTimer()
	lib := t.consensus.GetLIB().BlockNum()
	t.log.Debugf("[trx db] sync lib: %d \n", lib)
	process := &LibInfo{}
	t.outDb.First(process)
	// begin from lastLib + 1, thus each time update libinfo should be atomic
	lastLib := process.Lib + 1
	// be carefully, no where condition there !!
	// the reason is only one row in the table
	// if introduce the mechanism that record checkpoint, the where closure should be added
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
		tx := t.outDb.Begin()
		blockStart := common.EasyTimer()
		trxInfoList, _ := t.handleLibNotification(block)
		if trxInfoList != nil {
			for _, trxInfo := range trxInfoList {
				if err := tx.Create(trxInfo).Error; err != nil {
					t.log.Errorf("[trx db] when inserted block %d, error occurred: %v", block , err)
					tx.Rollback()
					return err
				}
			}
		}
		process.Lib = block
		process.LastCheckTime = time.Now().UTC().Unix()
		if err := tx.Save(process).Error; err != nil {
			tx.Rollback()
			t.log.Errorf("[trx db] when committed block %d, error occurred: %v", block , err)
		} else {
			tx.Commit()
		}
		t.log.Debugf("[trx db] insert block %d, spent: %s", block, blockStart)
	}
	t.log.Debugf("[trx db] PollLib spent: %v", start)
	return nil
}

func (t *TrxMysqlService) handleLibNotification(lib uint64) ([]*TrxInfo, error) {
	blks , err := t.consensus.FetchBlocks(lib, lib)
	if err != nil {
		t.log.Error(err)
		return nil, err
	}
	if len(blks) == 0 {
		return nil, nil
	}
	blk := blks[0].(*prototype.SignedBlock)
	var trxInfoList []*TrxInfo
	for _, trx := range blk.Transactions {
		trxHash, _ := trx.SigTrx.Id()
		trxId := hex.EncodeToString(trxHash.Hash)
		blockHeight := lib
		data := blk.Id().Data
		blockId := hex.EncodeToString(data[:])
		blockTime := blk.Timestamp()
		invoice, _ := json.Marshal(trx.Receipt)
		operations := PurgeOperation(trx.SigTrx.GetTrx().GetOperations())
		operationsJson, _ := json.Marshal(operations)
		creator := FindCreator(trx.SigTrx.GetTrx().GetOperations()[0])
		trxInfoList = append(trxInfoList,
			&TrxInfo{TrxId:trxId,
				BlockHeight:blockHeight,
				BlockId: blockId,
				BlockTime:blockTime,
				Invoice:invoice,
				Operations:operationsJson,
				Creator:creator})
	}
	return trxInfoList, nil
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