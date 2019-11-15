package plugins

import (
	"github.com/coschain/contentos-go/iservices"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

type SyncForwardManagerService struct {
	*ForwardManagerService
}

func NewSyncForwardManagerService(logger *logrus.Logger, db *gorm.DB, processors map[string]IBlockLogProcessor) *SyncForwardManagerService {
	return &SyncForwardManagerService{
		NewForwardManagerService(logger, db, processors, &SyncForwardMangerCheckpoint{db:db}),
	}
}

type SyncForwardMangerCheckpoint struct {
	db *gorm.DB
}

func (cp SyncForwardMangerCheckpoint) HasNeedSyncProcessors() bool {
	return true
}

func (cp SyncForwardMangerCheckpoint) ProgressesOfNeedSyncProcessors() []*iservices.Progress{
	var progresses []*iservices.Progress
	cp.db.Where(&iservices.Progress{FastForward:false}).Find(&progresses)
	return progresses
}

func (cp SyncForwardMangerCheckpoint) TryToTransferProcessorManager(progress *iservices.Progress) error {
	return nil
}
