package plugins

import (
	"github.com/coschain/contentos-go/app/blocklog"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"github.com/jinzhu/gorm"
	"time"
)

type OpProcessor func(operation prototype.BaseOperation, baseRecord interface{}) ([]interface{}, error)
type ChangeProcessor func(opType string, operation prototype.BaseOperation, change *blocklog.StateChange, baseRecord interface{}) ([]interface{}, error)

type OpProcessorManager struct {
	opProcessors map[string]OpProcessor
}

func (m *OpProcessorManager) Register(opType string, processor OpProcessor) {
	m.opProcessors[opType] = processor
}

func (m *OpProcessorManager) Find(opType string) (OpProcessor, bool) {
	processor, ok := m.opProcessors[opType]
	return processor, ok
}

type IOTrxProcessor struct {
	tableReady bool
	opProcessorManager *OpProcessorManager
	changeProcessors []ChangeProcessor
}

func NewIOTrxProcessor() *IOTrxProcessor{
	p := &IOTrxProcessor{opProcessorManager: &OpProcessorManager{}}
	p.registerOpProcessor()
	p.registerChangeProcessor()
	return p
}

func (p *IOTrxProcessor) registerOpProcessor() {
	p.opProcessorManager.Register("account_create", ProcessAccountCreateOperation)
	p.opProcessorManager.Register("account_update", ProcessAccountUpdateOperation)
	p.opProcessorManager.Register("acquire_ticket", ProcessAcquireTicketOperation)
	p.opProcessorManager.Register("bp_enable", ProcessBpEnableOperation)
	p.opProcessorManager.Register("bp_register", ProcessBpRegisterOperation)
	p.opProcessorManager.Register("bp_update", ProcessBpUpdateOperation)
	p.opProcessorManager.Register("bp_vote", ProcessBpVoteOperation)
	p.opProcessorManager.Register("contract_apply", ProcessContractApplyOperation)
	p.opProcessorManager.Register("contract_deploy", ProcessContractDeployOperation)
	p.opProcessorManager.Register("convert_vest", ProcessConvertVestOperation)
	p.opProcessorManager.Register("post", ProcessPostOperation)
	p.opProcessorManager.Register("reply", ProcessReplyOperation)
	p.opProcessorManager.Register("stake", ProcessStakeOperation)
	p.opProcessorManager.Register("transfer", ProcessTransferOperation)
	p.opProcessorManager.Register("transfer_to_vest", ProcessTransferVestOperation)
	p.opProcessorManager.Register("un_stake", ProcessUnStakeOperation)
	p.opProcessorManager.Register("vote", ProcessVoteOperation)
	p.opProcessorManager.Register("vote_by_ticket", ProcessVoteByTicketOperation)
}

func (p *IOTrxProcessor) registerChangeProcessor() {
	p.changeProcessors = append(p.changeProcessors,
		ProcessContractTransferToUserChangeProcessor,
		ProcessUserToContractChangeProcessor,
		ProcessContractTransferToContractChangeProcessor)
}

func (p *IOTrxProcessor) Prepare(db *gorm.DB, blockLog *blocklog.BlockLog) (err error) {
	if !p.tableReady {
		if !db.HasTable(&iservices.IOTrxRecord{}) {
			if err = db.CreateTable(&iservices.IOTrxRecord{}).Error; err == nil {
				p.tableReady = true
			}
		} else {
			p.tableReady = true
		}
	}
	return
}

func (p *IOTrxProcessor) ProcessChange(db *gorm.DB, change *blocklog.StateChange, blockLog *blocklog.BlockLog, changeIdx, opIdx, trxIdx int) error {
	trxLog := blockLog.Transactions[trxIdx]
	opLog := trxLog.Operations[opIdx]
	opType := opLog.Type
	op := prototype.GetBaseOperation(opLog.Data)
	record := iservices.IOTrxRecord{
		TrxHash:     trxLog.TrxId,
		BlockHeight: blockLog.BlockNum,
		BlockTime:   time.Unix(int64(blockLog.BlockTime), 0),
		Action:      opLog.Type,
	}
	for _, changeProcessor := range p.changeProcessors {
		records, err := changeProcessor(opType, op, change, record)
		if err != nil {
			return err
		}
		for _, record := range records {
			err := db.Create(record).Error
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *IOTrxProcessor) ProcessOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error {
	trxLog := blockLog.Transactions[trxIdx]
	opLog := trxLog.Operations[opIdx]
	opType := opLog.Type
	op := prototype.GetBaseOperation(opLog.Data)
	record := iservices.IOTrxRecord{
		TrxHash:     trxLog.TrxId,
		BlockHeight: blockLog.BlockNum,
		BlockTime:   time.Unix(int64(blockLog.BlockTime), 0),
		Action:      opLog.Type,
	}
	if processor, ok := p.opProcessorManager.Find(opType); ok {
		records, err := processor(op, record)
		if err != nil {
			return err
		}
		for _, record := range records {
			err := db.Create(record).Error
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *IOTrxProcessor) Finalize(db *gorm.DB, blockLog *blocklog.BlockLog) error {
	return nil
}

func init() {
	RegisterSQLTableNamePattern("iotrx_record")
}