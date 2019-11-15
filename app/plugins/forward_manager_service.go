package plugins

import (
	"github.com/coschain/contentos-go/app/blocklog"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/node"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"sync"
	"sync/atomic"
	"time"
)

type Checkpoint interface {
	HasNeedSyncProcessors() bool
	ProgressesOfNeedSyncProcessors() []*iservices.Progress
	TryToTransferProcessorManager(progress *iservices.Progress) error
}

type ForwardManagerService struct {
	sync.Mutex
	logger *logrus.Logger
	db *gorm.DB
	jobTimer *time.Timer
	stop int32
	working int32
	workStop *sync.Cond
	mainProcessors map[string]IBlockLogProcessor
	point Checkpoint
}

func NewForwardManagerService(logger *logrus.Logger, db *gorm.DB, processors map[string]IBlockLogProcessor, point Checkpoint) *ForwardManagerService {
	s := &ForwardManagerService{logger:logger, db: db, mainProcessors:processors, point: point}
	s.workStop = sync.NewCond(&s.Mutex)
	return s
}

func (s *ForwardManagerService) Start(node *node.Node) error  {
	if s.point.HasNeedSyncProcessors() {
		s.scheduleNextJob()
	}
	return nil
}

func (s *ForwardManagerService) Stop() error  {
	s.waitWorkDone()
	if s.db != nil {
		_ = s.db.Close()
	}
	s.db, s.stop, s.working = nil, 0, 0
	return nil
}

func (s *ForwardManagerService) scheduleNextJob() {
	s.jobTimer = time.AfterFunc(1*time.Second, s.work)
}

func (s *ForwardManagerService) waitWorkDone() {
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

func (s *ForwardManagerService) work() {
	const maxJobSize = 1000
	var (
		userBreak = false
		err error
	)
	atomic.StoreInt32(&s.working, 1)

	progresses := s.point.ProgressesOfNeedSyncProcessors()

	for _, progress := range progresses {
		if atomic.LoadInt32(&s.stop) != 0 {
			userBreak = true
			break
		}
		processor, ok := s.mainProcessors[progress.Processor]
		if !ok {
			continue
		}
		minBlockNum, maxBlockNum := progress.BlockHeight+1, progress.BlockHeight+maxJobSize
		for blockNum := minBlockNum; blockNum <= maxBlockNum; blockNum++ {
			if atomic.LoadInt32(&s.stop) != 0 {
				userBreak = true
				break
			}
			blockLogRec := &iservices.BlockLogRecord{BlockHeight: blockNum}
			if s.db.Where(&iservices.BlockLogRecord{BlockHeight: blockNum, Final: true}).First(blockLogRec).RecordNotFound() {
				break
			}
			blockLog := new(blocklog.BlockLog)
			if err = blockLog.FromJsonString(blockLogRec.JsonLog); err != nil {
				break
			}
			tx := s.db.Begin()
			userBreak, err = s.processLog(tx, blockLog, processor)
			if !userBreak && err == nil {
				progress.BlockHeight = blockNum
				progress.FinishAt = time.Now()
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
		if atomic.LoadInt32(&s.stop) != 0 {
			userBreak = true
			break
		}
		if err := s.point.TryToTransferProcessorManager(progress); err != nil {
			s.logger.Errorf("try to transfer processor manager error: %v", err)
		}
	}
	s.Lock()
	atomic.StoreInt32(&s.working, 0)
	if !userBreak {
		if s.point.HasNeedSyncProcessors() {
			s.scheduleNextJob()
		}
	}
	s.workStop.Signal()
	s.Unlock()
}

func (s *ForwardManagerService) processLog(db *gorm.DB, blockLog *blocklog.BlockLog, processor IBlockLogProcessor) (userBreak bool, err error) {
	userBreak, err = s.callProcessors(processor, func(processor IBlockLogProcessor) error {
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
			userBreak, err = s.callProcessors(processor, func(processor IBlockLogProcessor) error {
				return processor.ProcessOperation(db, blockLog, opIdx, trxIdx)
			})
			if ok = !userBreak && err == nil; !ok {
				break
			}
			for changeIdx, change := range opLog.Changes {
				userBreak, err = s.callProcessors(processor, func(processor IBlockLogProcessor) error {
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
			userBreak, err = s.callProcessors(processor, func(processor IBlockLogProcessor) error {
				return processor.ProcessChange(db, change, blockLog, changeIdx, -1, -1)
			})
			if ok = !userBreak && err == nil; !ok {
				break
			}
		}
	}
	if ok {
		userBreak, err = s.callProcessors(processor, func(processor IBlockLogProcessor) error {
			return processor.Finalize(db, blockLog)
		})
	}
	return
}

func (s *ForwardManagerService) callProcessors(processor IBlockLogProcessor, f func(IBlockLogProcessor)error) (userBreak bool, err error) {
	if atomic.LoadInt32(&s.stop) != 0 {
		userBreak = true
		return
	}
	if err = f(processor); err != nil {
		return
	}
	return
}

