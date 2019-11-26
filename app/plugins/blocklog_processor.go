package plugins

import (
	"github.com/coschain/contentos-go/app/blocklog"
	"github.com/jinzhu/gorm"
)

type BlockLogProcessor struct {
	processors []IBlockLogProcessor
}

func NewBlockLogProcessor() *BlockLogProcessor {
	blockLogProcessor := &BlockLogProcessor{}
	blockLogProcessor.processors = append(blockLogProcessor.processors,
		NewHolderProcessor(),
		NewStakeProcessor(),
		NewTransferProcessor(),
		NewCreateUserProcessor(),
		NewEcosysProcessor(),
		NewProducerVoteProcessor(),
		NewPowerUpDownProcessor(),
	)
	return blockLogProcessor
}

func (p *BlockLogProcessor) Prepare(db *gorm.DB, blockLog *blocklog.BlockLog) error {
	for _, processor := range p.processors {
		if err := processor.Prepare(db, blockLog); err != nil {
			return err
		}
	}
	return nil
}

func (p *BlockLogProcessor) ProcessChange(db *gorm.DB, change *blocklog.StateChange, blockLog *blocklog.BlockLog, changeIdx, opIdx, trxIdx int) error {
	for _, processor := range p.processors {
		if err := processor.ProcessChange(db, change, blockLog, changeIdx, opIdx, trxIdx); err != nil {
			return err
		}
	}
	return nil
}

func (p *BlockLogProcessor) ProcessOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error {
	for _, processor := range p.processors {
		if err := processor.ProcessOperation(db, blockLog, opIdx, trxIdx); err != nil {
			return err
		}
	}
	return nil
}

func (p *BlockLogProcessor) Finalize(db *gorm.DB, blockLog *blocklog.BlockLog) error {
	for _, processor := range p.processors {
		if err := processor.Finalize(db, blockLog); err != nil {
			return err
		}
	}
	return nil
}

