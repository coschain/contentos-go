package plugins

import (
	"github.com/coschain/contentos-go/app/blocklog"
	"github.com/coschain/contentos-go/iservices"
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

func NewForwardManagerService(logger *logrus.Logger, db *gorm.DB, processors map[string]IBlockLogProcessor) *ForwardManagerService {
	s := &ForwardManagerService{logger:logger, db: db, mainProcessors:processors}
	s.workStop = sync.NewCond(&s.Mutex)
	return s
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

