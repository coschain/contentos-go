package plugins

import (
	"encoding/json"
	"errors"
	"github.com/coschain/contentos-go/app/blocklog"
	"github.com/coschain/contentos-go/prototype"
	"github.com/jinzhu/gorm"
	"time"
)


type SbtRecord struct {
	ID uint64					`gorm:"primary_key;auto_increment"`
	BlockHeight uint64			`gorm:"index"`
	BlockTime time.Time
	Action string				`gorm:"index"`
	Symbol string				`gorm:"index:idx_symbol_token"`
	TokenId string				`gorm:"index:idx_symbol_token"`
	From string					`gorm:"index"`
	To string					`gorm:"index"`
}

type SbtState struct {
	ID uint64					`gorm:"primary_key;auto_increment"`
	Owner string				`gorm:"index"`
	Symbol string				`gorm:"index:idx_symbol_token"`
	TokenId string				`gorm:"index:idx_symbol_token"`
	BurnAuth string
}

type SbtTokenRecord struct {
	Symbol string				`json:"symbol"`
	Desc string					`json:"desc"`
	Uri string					`json:"uri"`
	MintedCount uint64			`json:"minted_count"`
	BurnedCount uint64			`json:"burned_count"`
	Issuer string				`json:"issuer"`
	IssuedAt uint64				`json:"issued_at"`
}

type SbtHoldingRecord struct {
	GlobalId string				`json:"global_id"`
	Symbol string				`json:"symbol"`
	Token string				`json:"token"`
	Owner string				`json:"owner"`
	BurnAuth string				`json:"burn_auth"`
}

type SbtProcessor struct {
	tableRecordReady bool
	tableStateReady bool
}

func NewSbtProcessor() *SbtProcessor {
	return &SbtProcessor{}
}

func (p *SbtProcessor) Prepare(db *gorm.DB, blockLog *blocklog.BlockLog) (err error) {
	if !p.tableRecordReady {
		if !db.HasTable(&SbtRecord{}) {
			if err = db.CreateTable(&SbtRecord{}).Error; err == nil {
				p.tableRecordReady = true
			}
		} else {
			p.tableRecordReady = true
		}
	}

	if !p.tableStateReady {
		if !db.HasTable(&SbtState{}) {
			if err = db.CreateTable(&SbtState{}).Error; err == nil {
				p.tableStateReady = true
			}
		} else {
			p.tableStateReady = true
		}
	}

	return
}

func (p *SbtProcessor) ProcessChange(db *gorm.DB, change *blocklog.StateChange, blockLog *blocklog.BlockLog, changeIdx, opIdx, trxIdx int) error {
	SbtContractOwner := "contentos"
	SbtContractName := "cossbt"
	SbtContract := "@" + SbtContractOwner + "." + SbtContractName
	SbtContractTokensTable := SbtContract + ".tokens"
	SbtContractHoldingsTable := SbtContract + ".holdings"

	if opIdx < 0 || trxIdx < 0 {
		return nil
	}
	trxLog := blockLog.Transactions[trxIdx]
	if trxLog.Receipt.Status != prototype.StatusSuccess {
		return nil
	}
	opLog := trxLog.Operations[opIdx]
	if opLog.Type != "contract_apply" {
		return nil
	}
	op, ok := prototype.GetBaseOperation(opLog.Data).(*prototype.ContractApplyOperation)
	if !ok {
		return errors.New("failed converting to ContractApplyOperation")
	}
	if op.Owner.Value != SbtContractOwner || op.Contract != SbtContractName {
		return nil
	}

	var (
		tokenRecord SbtTokenRecord
		holdingBefore, holdingAfter SbtHoldingRecord
		state *SbtState
		stateOp string
	)
	rec := &SbtRecord {
		BlockHeight: blockLog.BlockNum,
		BlockTime: time.Unix(int64(blockLog.BlockTime), 0),
		Action: op.Method,
	}

	if op.Method == "issue" && change.What == SbtContractTokensTable && change.Kind == blocklog.ChangeKindCreate {
		if err := p.parseRecord(change.Change.After, &tokenRecord); err != nil {
			return err
		}
		rec.Symbol = tokenRecord.Symbol
		rec.TokenId = ""
		rec.From = op.Caller.Value
		rec.To = ""
	} else if change.What == SbtContractHoldingsTable {
		if err := p.parseRecord(change.Change.Before, &holdingBefore); err != nil {
			return err
		}
		if err := p.parseRecord(change.Change.After, &holdingAfter); err != nil {
			return err
		}
		holdingRecord := &holdingAfter
		if op.Method == "mint" {
			rec.From = op.Caller.Value
			rec.To = holdingAfter.Owner
		} else if op.Method == "burn" {
			holdingRecord = &holdingBefore
			rec.From = op.Caller.Value
			rec.To = holdingBefore.Owner
		} else {
			return nil
		}
		rec.Symbol = holdingRecord.Symbol
		rec.TokenId = holdingRecord.Token
		state = &SbtState{
			Owner: holdingRecord.Owner,
			Symbol: holdingRecord.Symbol,
			TokenId: holdingRecord.Token,
			BurnAuth: holdingRecord.BurnAuth,
		}
		stateOp = change.Kind
	} else {
		return nil
	}
	if err := db.Create(rec).Error; err != nil {
		return err
	}
	if state != nil {
		if stateOp == blocklog.ChangeKindCreate {
			return db.Create(state).Error
		} else if stateOp == blocklog.ChangeKindDelete {
			return db.Where("symbol = ? AND token_id = ?", state.Symbol, state.TokenId).Delete(&SbtState{}).Error
		} else if stateOp == blocklog.ChangeKindUpdate {
			// this should never be reached
			return db.Model(&SbtState{}).Where("symbol = ? AND token_id = ?", state.Symbol, state.TokenId).Update("owner", state.Owner).Error
		}
	}
	return nil
}

func (p *SbtProcessor) ProcessOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error {
	return nil
}

func (p *SbtProcessor) Finalize(db *gorm.DB, blockLog *blocklog.BlockLog) error {
	return nil
}

func (p *SbtProcessor) parseRecord(obj interface{}, output interface{}) error {
	if jsonBytes, err := json.Marshal(obj); err != nil {
		return err
	} else {
		return json.Unmarshal(jsonBytes, output)
	}
}


func init() {
	RegisterSQLTableNamePattern("sbt_records")
	RegisterSQLTableNamePattern("sbt_states")
}
