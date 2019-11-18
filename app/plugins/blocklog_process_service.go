package plugins

import (
	"fmt"
	"github.com/coschain/contentos-go/app/blocklog"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/iservices"
	service_configs "github.com/coschain/contentos-go/iservices/service-configs"
	"github.com/coschain/contentos-go/node"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type IBlockLogProcessor interface {
	Prepare(db *gorm.DB, blockLog *blocklog.BlockLog) error
	ProcessChange(db *gorm.DB, change *blocklog.StateChange, blockLog *blocklog.BlockLog, changeIdx, opIdx, trxIdx int) error
	ProcessOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error
	Finalize(db *gorm.DB, blockLog *blocklog.BlockLog) error
}

type BlockLogProcessService struct {
	sync.Mutex
	config *service_configs.DatabaseConfig
	logger *logrus.Logger
	db *gorm.DB
	ctx *node.ServiceContext
	fastForwardService node.Service
	syncForwardService node.Service
	processors map[string]IBlockLogProcessor
}

func NewBlockLogProcessService(ctx *node.ServiceContext, config *service_configs.DatabaseConfig, log *logrus.Logger) (*BlockLogProcessService, error) {
	processors := make(map[string]IBlockLogProcessor)
	return &BlockLogProcessService{ctx: ctx, logger: log, config: config, processors:processors}, nil
}

func (s *BlockLogProcessService) Start(node *node.Node) error {
	s.register("blocklog", NewBlockLogProcessor())
	s.register("iotrx", NewIOTrxProcessor())
	s.register("ecosys_powerdown", NewEcosysPowerDownProcessor())
	if err := s.initDatabase(); err != nil {
		return fmt.Errorf("invalid database: %s", err)
	}
	s.fastForwardService = NewFastForwardManagerService(s.logger, s.db, s.processors)
	s.syncForwardService = NewSyncForwardManagerService(s.logger, s.db, s.processors)
	go s.fastForwardService.Start(node)
	go s.syncForwardService.Start(node)
	return nil
}

func (s *BlockLogProcessService) Stop() error {
	_ = s.fastForwardService.Stop()
	_ = s.syncForwardService.Stop()
	if s.db != nil {
		_ = s.db.Close()
	}
	return nil
}

func (s *BlockLogProcessService) register(name string, processor IBlockLogProcessor) {
	s.processors[name] = processor
}

// hard code is ok there.
func (s *BlockLogProcessService) migrateDeprecatedProgress() {
	deprecatedProgress := &iservices.DeprecatedBlockLogProgress{}
	if s.db.HasTable(deprecatedProgress) {
		if !s.db.First(deprecatedProgress).RecordNotFound() {
			progress := &iservices.Progress{}
			s.db.Where(&iservices.Progress{Processor: "blocklog"}).First(progress)
			if deprecatedProgress.BlockHeight > progress.BlockHeight {
				progress.BlockHeight = deprecatedProgress.BlockHeight
				progress.FinishAt = deprecatedProgress.FinishAt
				s.db.Save(progress)
			}
		}
	}
}

func (s *BlockLogProcessService) initDatabase() error {
	connStr := fmt.Sprintf("%s:%s@/%s?charset=utf8mb4&parseTime=True&loc=Local", s.config.User, s.config.Password, s.config.Db)
	if db, err := gorm.Open(s.config.Driver, connStr); err != nil {
		return err
	} else {
		s.db = db
	}
	if !s.db.HasTable(&iservices.Progress{}) {
		if err := s.db.CreateTable(&iservices.Progress{}).Error; err != nil {
			_ = s.db.Close()
			return err
		}
	}
	for k := range s.processors {
		progress := &iservices.Progress{}
		if s.db.Where(&iservices.Progress{Processor: k}).First(progress).RecordNotFound() {
			progress.Processor = k
			progress.BlockHeight = 0
			fastForward := true
			progress.FastForward = &fastForward
			progress.FinishAt = time.Unix(constants.GenesisTime, 0)
			if err := s.db.Create(progress).Error; err != nil {
				return err
			}
		}
	}
	s.migrateDeprecatedProgress()
	return nil
}

func init() {
	RegisterSQLTableNamePattern("progress")
}