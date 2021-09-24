package plugins

import (
	"encoding/json"
	"errors"
	"github.com/coschain/contentos-go/app/blocklog"
	"github.com/coschain/contentos-go/prototype"
	"github.com/jinzhu/gorm"
	"time"
)


type NftRecord struct {
	ID uint64					`gorm:"primary_key;auto_increment"`
	BlockHeight uint64			`gorm:"index"`
	BlockTime time.Time
	Action string				`gorm:"index"`
	Symbol string				`gorm:"index:idx_symbol_token;index:idx_symbol"`
	TokenId string				`gorm:"index:idx_symbol_token"`
	From string					`gorm:"index"`
	To string					`gorm:"index"`
}

type NftState struct {
	ID uint64					`gorm:"primary_key;auto_increment"`
	Owner string				`gorm:"index"`
	Symbol string				`gorm:"index:idx_symbol_token"`
	TokenId string				`gorm:"index:idx_symbol_token"`
}

type NftTokenRecord struct {
	Symbol string				`json:"symbol"`
	Desc string					`json:"desc"`
	Uri string					`json:"uri"`
	MintedCount uint64			`json:"minted_count"`
	BurnedCount uint64			`json:"burned_count"`
	TransferredCount uint64		`json:"transferred_count"`
	Issuer string				`json:"issuer"`
	IssuedAt uint64				`json:"issued_at"`
}

type NftHoldingRecord struct {
	GlobalId string				`json:"global_id"`
	Symbol string				`json:"symbol"`
	Token string				`json:"token"`
	Owner string				`json:"owner"`
}

type NftProcessor struct {
	tableRecordReady bool
	tableStateReady bool
}

func NewNftProcessor() *NftProcessor {
	return &NftProcessor{}
}

func (p *NftProcessor) Prepare(db *gorm.DB, blockLog *blocklog.BlockLog) (err error) {
	if !p.tableRecordReady {
		if !db.HasTable(&NftRecord{}) {
			if err = db.CreateTable(&NftRecord{}).Error; err == nil {
				p.tableRecordReady = true
			}
		} else {
			p.tableRecordReady = true
		}
	}

	if !p.tableStateReady {
		if !db.HasTable(&NftState{}) {
			if err = db.CreateTable(&NftState{}).Error; err == nil {
				p.tableStateReady = true
			}
		} else {
			p.tableStateReady = true
		}
	}

	return
}

func (p *NftProcessor) ProcessChange(db *gorm.DB, change *blocklog.StateChange, blockLog *blocklog.BlockLog, changeIdx, opIdx, trxIdx int) error {
	NftContractOwner := "contentos"
	NftContractName := "cosnft"
	NftContract := "@" + NftContractOwner + "." + NftContractName
	NftContractTokensTable := NftContract + ".tokens"
	NftContractHoldingsTable := NftContract + ".holdings"

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
	if op.Owner.Value != NftContractOwner || op.Contract != NftContractName {
		return nil
	}

	var (
		tokenRecord NftTokenRecord
		holdingBefore, holdingAfter NftHoldingRecord
		state *NftState
		stateOp string
	)
	rec := &NftRecord {
		BlockHeight: blockLog.BlockNum,
		BlockTime: time.Unix(int64(blockLog.BlockTime), 0),
		Action: op.Method,
	}

	if op.Method == "issue" && change.What == NftContractTokensTable && change.Kind == blocklog.ChangeKindCreate {
		if err := p.parseRecord(change.Change.After, &tokenRecord); err != nil {
			return err
		}
		rec.Symbol = tokenRecord.Symbol
		rec.TokenId = ""
		rec.From = op.Caller.Value
		rec.To = ""
	} else if change.What == NftContractHoldingsTable {
		if err := p.parseRecord(change.Change.Before, &holdingBefore); err != nil {
			return err
		}
		if err := p.parseRecord(change.Change.After, &holdingAfter); err != nil {
			return err
		}
		holdingRecord := &holdingAfter
		if op.Method == "mint" {
			rec.From = ""
			rec.To = holdingAfter.Owner
		} else if op.Method == "burn" {
			holdingRecord = &holdingBefore
			rec.From = holdingBefore.Owner
			rec.To = ""
		} else if op.Method == "transfer" {
			rec.From = holdingBefore.Owner
			rec.To = holdingAfter.Owner
		} else {
			return nil
		}
		rec.Symbol = holdingRecord.Symbol
		rec.TokenId = holdingRecord.Token
		state = &NftState{
			Owner: holdingRecord.Owner,
			Symbol: holdingRecord.Symbol,
			TokenId: holdingRecord.Token,
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
			return db.Where("symbol = ? AND token_id = ?", state.Symbol, state.TokenId).Delete(&NftState{}).Error
		} else if stateOp == blocklog.ChangeKindUpdate {
			return db.Model(&NftState{}).Where("symbol = ? AND token_id = ?", state.Symbol, state.TokenId).Update("owner", state.Owner).Error
		}
	}
	return nil
}

func (p *NftProcessor) ProcessOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error {
	return nil
}

func (p *NftProcessor) Finalize(db *gorm.DB, blockLog *blocklog.BlockLog) error {
	return nil
}

func (p *NftProcessor) parseRecord(obj interface{}, output interface{}) error {
	if jsonBytes, err := json.Marshal(obj); err != nil {
		return err
	} else {
		return json.Unmarshal(jsonBytes, output)
	}
}


func init() {
	RegisterSQLTableNamePattern("nft_records")
	RegisterSQLTableNamePattern("nft_states")
}
