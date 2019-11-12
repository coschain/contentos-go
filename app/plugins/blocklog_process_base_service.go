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

type ProgressBuilder func() IProcess

type IProcess interface {
	GetBlockHeight() uint64
	SetBlockHeight(blockHeight uint64)
	SetFinishAt(finishAt time.Time)
}

type IBlockLogProcessor interface {
	Prepare(db *gorm.DB, blockLog *blocklog.BlockLog) error
	ProcessChange(db *gorm.DB, change *blocklog.StateChange, blockLog *blocklog.BlockLog, changeIdx, opIdx, trxIdx int) error
	ProcessOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error
	Finalize(db *gorm.DB, blockLog *blocklog.BlockLog) error
}

type BlockLogProcessBaseService struct {
	sync.Mutex
	config *service_configs.DatabaseConfig
	logger *logrus.Logger
	db *gorm.DB
	jobTimer *time.Timer
	stop int32
	working int32
	workStop *sync.Cond
	progressBuilder ProgressBuilder
	processors []IBlockLogProcessor
}

func NewBlockLogProcessBaseService(ctx *node.ServiceContext, config *service_configs.DatabaseConfig, logger *logrus.Logger,
	processBuilder ProgressBuilder ) (*BlockLogProcessBaseService, error) {
	s := &BlockLogProcessBaseService{ config:config, logger:logger }
	s.workStop = sync.NewCond(&s.Mutex)
	s.progressBuilder = processBuilder
	return s, nil
}

func (s *BlockLogProcessBaseService) Start(node *node.Node) error  {
	if err := s.initDatabase(); err != nil {
		return fmt.Errorf("invalid database: %s", err.Error())
	}
	s.scheduleNextJob()
	return nil
}

func (s *BlockLogProcessBaseService) Stop() error  {
	s.waitWorkDone()
	if s.db != nil {
		_ = s.db.Close()
	}
	s.db, s.stop, s.working = nil, 0, 0
	return nil
}

func (s *BlockLogProcessBaseService) initDatabase() error {
	connStr := fmt.Sprintf("%s:%s@/%s?charset=utf8mb4&parseTime=True&loc=Local", s.config.User, s.config.Password, s.config.Db)
	if db, err := gorm.Open(s.config.Driver, connStr); err != nil {
		return err
	} else {
		s.db = db
	}
	progress := s.progressBuilder()
	progress.SetBlockHeight(0)
	progress.SetFinishAt(time.Now())
	if !s.db.HasTable(progress) {
		if err := s.db.CreateTable(progress).Error; err != nil {
			_ = s.db.Close()
			return err
		}
		s.db.Create(progress)
	}
	return nil
}

func (s *BlockLogProcessBaseService) scheduleNextJob() {
	s.jobTimer = time.AfterFunc(1 * time.Second, s.work)
}

func (s *BlockLogProcessBaseService) waitWorkDone() {
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

func (s *BlockLogProcessBaseService) work() {
	const maxJobSize = 1000
	var (
		userBreak = false
		err error
	)
	atomic.StoreInt32(&s.working, 1)

	progress := s.progressBuilder()
	s.db.First(progress)

	minBlockNum, maxBlockNum := progress.(IProcess).GetBlockHeight() + 1, progress.(IProcess).GetBlockHeight() + maxJobSize
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
			progress.SetBlockHeight(blockNum)
			progress.SetFinishAt(time.Now())
			if err = tx.Save(progress).Error; err == nil {
				tx.Commit()
			} else {
				s.logger.Errorf("save service progress failed and rolled back, error: %v", err)
				tx.Rollback()
				break
			}
		} else {
			s.logger.Errorf("process log failed and rolled back, error: %v", err)
			tx.Rollback()
			break
		}
	}
	s.Lock()
	atomic.StoreInt32(&s.working, 0)
	if !userBreak {
		s.scheduleNextJob()
	}
	s.workStop.Signal()
	s.Unlock()
}

func (s *BlockLogProcessBaseService) processLog(db *gorm.DB, blockLog *blocklog.BlockLog) (userBreak bool, err error) {
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

func (s *BlockLogProcessBaseService) callProcessors(f func(IBlockLogProcessor)error) (userBreak bool, err error) {
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
