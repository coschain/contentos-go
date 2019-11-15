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
	progress := &iservices.Progress{}
	fastForward := true
	notFound := cp.db.Where(&iservices.Progress{FastForward:&fastForward}).First(progress).RecordNotFound()
	return !notFound
}

func (cp FastForwardManagerCheckpoint) ProgressesOfNeedSyncProcessors() []*iservices.Progress {
	var progresses []*iservices.Progress
	fastForward := true
	cp.db.Where(&iservices.Progress{FastForward:&fastForward}).Find(&progresses)
	return progresses
}

func (cp FastForwardManagerCheckpoint) TryToTransferProcessorManager(progress *iservices.Progress) error {
	blogLog := &iservices.BlockLogRecord{}
	cp.db.Last(blogLog)
	if *progress.FastForward == true {
		if blogLog.BlockHeight - progress.BlockHeight < uint64(iservices.ThresholdForFastConvertToSync) {
			fastForward := false
			progress.FastForward = &fastForward
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

