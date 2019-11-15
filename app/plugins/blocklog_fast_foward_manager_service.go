package plugins

import (
	"github.com/coschain/contentos-go/iservices"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

type FastForwardManagerService struct {
	*ForwardManagerService
}

func NewFastForwardManagerService(logger *logrus.Logger, db *gorm.DB, processors map[string]IBlockLogProcessor) *FastForwardManagerService {
	return &FastForwardManagerService{
		NewForwardManagerService(logger, db, processors, &FastForwardManagerCheckpoint{db:db}),
	}
}

type FastForwardManagerCheckpoint struct {
	db *gorm.DB
}

func (cp FastForwardManagerCheckpoint) HasNeedSyncProcessors() bool {
	var progresses []*iservices.Progress
	return cp.db.Where(&iservices.Progress{FastForward:true}).Find(&progresses).RecordNotFound()
}

func (cp FastForwardManagerCheckpoint) ProgressesOfNeedSyncProcessors() []*iservices.Progress {
	var progresses []*iservices.Progress
	cp.db.Where(&iservices.Progress{FastForward:true}).Find(&progresses)
	return progresses
}

func (cp FastForwardManagerCheckpoint) TryToTransferProcessorManager(progress *iservices.Progress) error {
	blogLog := &iservices.BlockLogRecord{}
	cp.db.Last(blogLog)
	if progress.FastForward == true {
		if blogLog.BlockHeight - progress.BlockHeight < iservices.ThresholdForFastConvertToSync {
			progress.FastForward = false
			tx := cp.db.Begin()
			if err := tx.Save(progress).Error; err == nil {
				tx.Commit()
			} else {
				tx.Rollback()
				return err
			}
		}
	}
	return nil
}

