package plugins

import (
	service_configs "github.com/coschain/contentos-go/iservices/service-configs"
	"github.com/coschain/contentos-go/node"
	"github.com/sirupsen/logrus"
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

func (p IOTrxProcess) GetBlockHeight() uint64 {
	return p.BlockHeight
}

func (p *IOTrxProcess) SetBlockHeight(blockHeight uint64) {
	p.BlockHeight = blockHeight
}

func (p *IOTrxProcess) SetFinishAt(finishAt time.Time) {
	p.FinishAt = finishAt
}

type IOTrxService struct {
	*BlockLogProcessBaseService
}

func NewIOTrxService(ctx *node.ServiceContext, config *service_configs.DatabaseConfig, logger *logrus.Logger) (*IOTrxService, error) {
	baseService, err := NewBlockLogProcessBaseService(ctx, config, logger, func() IProcess {
		return &IOTrxProcess{}
	})
	if err != nil {
		return nil, err
	}
	baseService.processors = append(baseService.processors,
		NewIOTrxProcessor(),
	)
	return &IOTrxService{BlockLogProcessBaseService: baseService}, err
}

func init() {
	RegisterSQLTableNamePattern("iotrx_process")
}