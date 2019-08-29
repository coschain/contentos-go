package plugins

import (
	"errors"
	"github.com/coschain/contentos-go/app/blocklog"
	"github.com/coschain/contentos-go/prototype"
	"github.com/jinzhu/gorm"
	"time"
)

type ProducerVoteRecord struct {
	ID uint64					`gorm:"primary_key;auto_increment"`
	BlockHeight uint64			`gorm:"index"`
	BlockTime time.Time
	Voter string				`gorm:"index"`
	Producer string				`gorm:"index"`
	Cancel bool
}

type ProducerVoteState struct {
	Voter string				`gorm:"primary_key"`
	Producer string				`gorm:"index"`
}

type ProducerVoteProcessor struct {
	tableRecordReady bool
	tableStateReady bool
}

func NewProducerVoteProcessor() *ProducerVoteProcessor {
	return &ProducerVoteProcessor{}
}

func (p *ProducerVoteProcessor) Prepare(db *gorm.DB, blockLog *blocklog.BlockLog) (err error) {
	if !p.tableRecordReady {
		if !db.HasTable(&ProducerVoteRecord{}) {
			if err = db.CreateTable(&ProducerVoteRecord{}).Error; err == nil {
				p.tableRecordReady = true
			}
		} else {
			p.tableRecordReady = true
		}
	}

	if !p.tableStateReady {
		if !db.HasTable(&ProducerVoteState{}) {
			if err = db.CreateTable(&ProducerVoteState{}).Error; err == nil {
				p.tableStateReady = true
			}
		} else {
			p.tableStateReady = true
		}
	}

	return
}

func (p *ProducerVoteProcessor) ProcessChange(db *gorm.DB, change *blocklog.StateChange, blockLog *blocklog.BlockLog, changeIdx, opIdx, trxIdx int) error {
	return nil
}

func (p *ProducerVoteProcessor) ProcessOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error {
	trxLog := blockLog.Transactions[trxIdx]

	if trxLog.Receipt.Status != prototype.StatusSuccess {
		return nil
	}
	opLog := trxLog.Operations[opIdx]
	if opLog.Type != "bp_vote" {
		return nil
	}

	op, ok := prototype.GetBaseOperation(opLog.Data).(*prototype.BpVoteOperation)
	if !ok {
		return errors.New("failed conversion to BpVoteOperation")
	}
	err := db.Create(&ProducerVoteRecord{
		BlockHeight: blockLog.BlockNum,
		BlockTime: time.Unix(int64(blockLog.BlockTime), 0),
		Voter: op.GetVoter().GetValue(),
		Producer: op.GetBlockProducer().GetValue(),
		Cancel: op.Cancel,
	}).Error

	if err != nil {
		return err
	}

	state := ProducerVoteState{ Voter: op.GetVoter().Value, Producer: op.GetBlockProducer().Value }

	if op.Cancel {
		return db.Delete( &state ).Error
	} else {
		return db.Create(&state).Error
	}

	return nil
}

func (p *ProducerVoteProcessor) Finalize(db *gorm.DB, blockLog *blocklog.BlockLog) error {
	return nil
}


func init() {
	RegisterSQLTableNamePattern("producer_vote_records")
	RegisterSQLTableNamePattern("producer_vote_states")
}
