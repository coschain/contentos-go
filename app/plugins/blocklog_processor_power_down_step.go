package plugins

import (
	"encoding/json"
	"fmt"
	"github.com/coschain/contentos-go/app/blocklog"
	"github.com/coschain/contentos-go/common"
	"github.com/jinzhu/gorm"
	"time"
)

type EcosysPowerDown struct {
	ID uint64					`gorm:"primary_key;auto_increment"`
	BlockHeight uint64			`gorm:"index"`
	BlockTime time.Time
	Name string					`gorm:"index"`
	VestOld uint64
	VestNew uint64
}

const PowerDownTableName = "ecosys_powerdown"

func (rec *EcosysPowerDown) TableName() string {
	num := rec.BlockHeight
	if num <= 0 {
		return PowerDownTableName
	}
	c := num / 5000000

	return fmt.Sprintf("%s_%d", PowerDownTableName, c )
}

type EcosysPowerDownProcessor struct {
	tableReady map[string]bool
}

func NewEcosysPowerDownProcessor() *EcosysPowerDownProcessor {
	r := &EcosysPowerDownProcessor{}
	r.tableReady = make(map[string]bool)
	return r
}

func (p *EcosysPowerDownProcessor) Prepare(db *gorm.DB, blockLog *blocklog.BlockLog) (err error) {
	rec := &EcosysPowerDown{ BlockHeight:blockLog.BlockNum }
	tableName := rec.TableName()

	if !p.tableReady[tableName] {
		if !db.HasTable(rec) {
			if err = db.CreateTable(rec).Error; err == nil {
				p.tableReady[tableName] = true
			}
		} else {
			p.tableReady[tableName] = true
		}
	}
	return
}

func (p *EcosysPowerDownProcessor) ProcessChange(db *gorm.DB, change *blocklog.StateChange, blockLog *blocklog.BlockLog, changeIdx, opIdx, trxIdx int) error {
	if change.What != "Account.Vest" {
		return nil
	}
	if change.Cause == "esys.power_down" {
		after := common.JsonNumberUint64(change.Change.After.(json.Number))
		before := common.JsonNumberUint64(change.Change.Before.(json.Number))

		return db.Create(&EcosysPowerDown{
			BlockHeight: blockLog.BlockNum,
			BlockTime: time.Unix(int64(blockLog.BlockTime), 0),
			Name: change.Change.Id.(string),
			VestOld: before,
			VestNew: after,
		}).Error
	}
	return nil
}

func (p *EcosysPowerDownProcessor) ProcessOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error {
	return nil
}

func (p *EcosysPowerDownProcessor) Finalize(db *gorm.DB, blockLog *blocklog.BlockLog) error {
	return nil
}

func init() {
	RegisterSQLTableNamePattern(fmt.Sprintf("%s\\w*", PowerDownTableName))
}
