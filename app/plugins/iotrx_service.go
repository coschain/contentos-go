package plugins

import (
	"fmt"
	"github.com/coschain/contentos-go/app/blocklog"
	"github.com/coschain/contentos-go/iservices"
	service_configs "github.com/coschain/contentos-go/iservices/service-configs"
	"github.com/coschain/contentos-go/node"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"sync"
	"sync/atomic"
	"time"
)

type IOTrxProcess struct {
	ID 				uint64			`gorm:"primary_key;auto_increment"`
	BlockHeight 	uint64
	FinishAt 		time.Time
}

func (IOTrxProcess) TableName() string {
	return "iotrx_process"
}

type IOTrxService struct {
	sync.Mutex
	config *service_configs.DatabaseConfig
	log *logrus.Logger
	db *gorm.DB
	jobTimer *time.Timer
	stop int32
	working int32
	workStop *sync.Cond
	processors []IBlockLogProcessor
}

func NewIOTrxService(ctx *node.ServiceContext, config *service_configs.DatabaseConfig, log *logrus.Logger) (*IOTrxService, error) {
	s := &IOTrxService{log: log, config: config}
	s.workStop = sync.NewCond(&s.Mutex)
	return s, nil
}

func (s *IOTrxService) Start(node *node.Node) error {
	connStr := fmt.Sprintf("%s:%s@/%s?charset=utf8mb4&parseTime=True&loc=Local", s.config.User, s.config.Password, s.config.Db)
	if db, err := gorm.Open(s.config.Driver, connStr); err != nil {
		return err
	} else {
		s.db = db
	}
	progress := &IOTrxProcess{
		BlockHeight: 0,
		FinishAt: time.Now(),
	}
	if !s.db.HasTable(progress) {
		if err := s.db.CreateTable(progress).Error; err != nil {
			_ = s.db.Close()
			return err
		}
		s.db.Create(progress)
	}
	s.addProcessors()
	s.scheduleNextJob()
	return nil
}

func (s *IOTrxService) addProcessors() {
	s.processors = append(s.processors,
		NewIOTrxProcessor(),
	)
}

func (s *IOTrxService) Stop() error {
	s.waitWorkDone()
	if s.db != nil {
		_ = s.db.Close()
	}
	s.db, s.stop, s.working = nil, 0, 0
	return nil
}

func (s *IOTrxService) scheduleNextJob() {
	s.jobTimer = time.AfterFunc(1 * time.Second, s.work)
}

func (s *IOTrxService) waitWorkDone() {
	s.Lock()
	if s.jobTimer != nil {
		s.jobTimer.Stop()
	}
	atomic.StoreInt32(&s.stop, 1)
	for atomic.LoadInt32(&s.working) != 0 {
		s.workStop.Wait()
	}
	s.Unlock()
}

func (s *IOTrxService) work() {
	const maxJobSize = 1000
	var (
		userBreak = false
		err error
	)
	atomic.StoreInt32(&s.working, 1)
	progress := &IOTrxProcess{}
	s.db.First(progress)

	minBlockNum, maxBlockNum := progress.BlockHeight + 1, progress.BlockHeight + maxJobSize
	for blockNum := minBlockNum; blockNum <= maxBlockNum; blockNum++ {
		if atomic.LoadInt32(&s.stop) != 0 {
			userBreak = true
			break
		}
		blockLogRec := &iservices.BlockLogRecord{ BlockHeight:blockNum }
		if s.db.Where(&iservices.BlockLogRecord{BlockHeight:blockNum, Final:true}).First(blockLogRec).RecordNotFound() {
			break
		}
		blockLog := new(blocklog.BlockLog)
		if err = blockLog.FromJsonString(blockLogRec.JsonLog); err != nil {
			break
		}
		tx := s.db.Begin()
		userBreak, err = s.processLog(tx, blockLog)
		if !userBreak && err == nil {
			progress.BlockHeight = blockNum
			progress.FinishAt = time.Now()
			if err = tx.Save(progress).Error; err == nil {
				tx.Commit()
			} else {
				tx.Rollback()
				break
			}
		} else {
			tx.Rollback()
			break
		}
	}

	s.Lock()
	atomic.StoreInt32(&s.working, 0)
	if atomic.LoadInt32(&s.stop) == 0 {
		s.scheduleNextJob()
	}
	s.workStop.Signal()
	s.Unlock()
}

func (s *IOTrxService) processLog(db *gorm.DB, blockLog *blocklog.BlockLog) (userBreak bool, err error) {
	userBreak, err = s.callProcessors(func(processor IBlockLogProcessor) error {
		return processor.Prepare(db, blockLog)
	})
	if userBreak || err != nil {
		return
	}
	ok := true
	for trxIdx, trxLog := range blockLog.Transactions {
		if !ok {
			break
		}
		for opIdx, opLog := range trxLog.Operations {
			if !ok {
				break
			}
			userBreak, err = s.callProcessors(func(processor IBlockLogProcessor) error {
				return processor.ProcessOperation(db, blockLog, opIdx, trxIdx)
			})
			if ok = !userBreak && err == nil; !ok {
				break
			}
			for changeIdx, change := range opLog.Changes {
				userBreak, err = s.callProcessors(func(processor IBlockLogProcessor) error {
					return processor.ProcessChange(db, change, blockLog, changeIdx, opIdx, trxIdx)
				})
				if ok = !userBreak && err == nil; !ok {
					break
				}
			}
		}
	}
	if ok {
		for changeIdx, change := range blockLog.Changes {
			userBreak, err = s.callProcessors(func(processor IBlockLogProcessor) error {
				return processor.ProcessChange(db, change, blockLog, changeIdx, -1, -1)
			})
			if ok = !userBreak && err == nil; !ok {
				break
			}
		}
	}
	if ok {
		userBreak, err = s.callProcessors(func(processor IBlockLogProcessor) error {
			return processor.Finalize(db, blockLog)
		})
	}
	return
}

func (s *IOTrxService) callProcessors(f func(IBlockLogProcessor)error) (userBreak bool, err error) {
	for _, processor := range s.processors {
		if atomic.LoadInt32(&s.stop) != 0 {
			userBreak = true
			break
		}
		if err = f(processor); err != nil {
			break
		}
	}
	return
}

func init() {
	RegisterSQLTableNamePattern("iotrx_process")
}