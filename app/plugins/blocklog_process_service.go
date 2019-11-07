package plugins

import (
	"github.com/coschain/contentos-go/iservices/service-configs"
	"github.com/coschain/contentos-go/node"
	"github.com/sirupsen/logrus"
	"time"
)

type BlockLogProcess struct {
	ID 				uint64			`gorm:"primary_key;auto_increment"`
	BlockHeight 	uint64
	FinishAt 		time.Time
}

func (BlockLogProcess) TableName() string {
	return "blocklog_process"
}

func (p BlockLogProcess) GetBlockHeight() uint64 {
	return p.BlockHeight
}

func (p *BlockLogProcess) SetBlockHeight(blockHeight uint64) {
	p.BlockHeight = blockHeight
}

func (p *BlockLogProcess) SetFinishAt(finishAt time.Time) {
	p.FinishAt = finishAt
}

type BlockLogProcessService struct {
	*BlockLogProcessBaseService
}

func NewBlockLogProcessService(ctx *node.ServiceContext, config *service_configs.DatabaseConfig, logger *logrus.Logger) (*BlockLogProcessService, error) {
	baseService, err := NewBlockLogProcessBaseService(ctx, config, logger, func() IProcess {
		return &BlockLogProcess{}
	})
	if err != nil {
		return nil, err
	}
	baseService.processors = append(baseService.processors,
		NewHolderProcessor(),
		NewStakeProcessor(),
		NewTransferProcessor(),
		NewCreateUserProcessor(),
		NewEcosysProcessor(),
		NewProducerVoteProcessor(),
		NewPowerUpDownProcessor(),
	)
	return &BlockLogProcessService{BlockLogProcessBaseService: baseService}, err
}

func init() {
	RegisterSQLTableNamePattern("blocklog_process")
}
