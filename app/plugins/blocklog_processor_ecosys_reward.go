package plugins

import (
	"encoding/json"
	"fmt"
	"github.com/coschain/contentos-go/app/blocklog"
	"github.com/coschain/contentos-go/common"
	"github.com/jinzhu/gorm"
	"time"
)

type EcosysReward struct {
	ID uint64					`gorm:"primary_key;auto_increment"`
	BlockHeight uint64			`gorm:"index"`
	BlockTime time.Time
	Name string					`gorm:"index"`
	Type string					`gorm:"index"`
	PostId uint64				`gorm:"index"`
	Reward uint64
	Wvps string
	TotalWvps string
	Pool uint64
}


func (rec *EcosysReward) TableName() string {
	const RewardTableName = "ecosys_reward"

	num := rec.BlockHeight
	if num <= 0 {
		return RewardTableName
	}
	c := num / 5000000

	return fmt.Sprintf("%s_%d", RewardTableName, c )
}

type EcosysProcessor struct {
	tableReady map[string]bool
}

func NewEcosysProcessor() *EcosysProcessor {
	r := &EcosysProcessor{}
	r.tableReady = make(map[string]bool)
	return r
}

func (p *EcosysProcessor) Prepare(db *gorm.DB, blockLog *blocklog.BlockLog) (err error) {
	rec := &EcosysReward{ BlockHeight:blockLog.BlockNum }
	tableName := rec.TableName()

	if !p.tableReady[tableName] {
		if !db.HasTable(rec) {
			if err = db.CreateTable(rec).Error; err == nil {
				p.tableReady[tableName] = true
			}
		}
	}
	return
}

func (p *EcosysProcessor) ProcessChange(db *gorm.DB, change *blocklog.StateChange, blockLog *blocklog.BlockLog, changeIdx, opIdx, trxIdx int) error {
	if change.Cause == "esys.reward.voter" ||
		change.Cause == "esys.reward.post_author" ||
		change.Cause == "esys.reward.reply_author" ||
		change.Cause == "esys.reward.dapp" {
		after := uint64(common.JsonNumber(change.Change.After.(json.Number)))
		before := uint64(common.JsonNumber(change.Change.Before.(json.Number)))
		postId := uint64(common.JsonNumberUint64(change.CauseExtra["post"].(json.Number)))
		pool := uint64(common.JsonNumberUint64(change.CauseExtra["pool"].(json.Number)))

		return db.Create(&EcosysReward{
			BlockHeight: blockLog.BlockNum,
			BlockTime: time.Unix(int64(blockLog.BlockTime), 0),
			Name: change.Change.Id.(string),
			Type: change.Cause,
			PostId: postId,
			Reward: after - before,
			Wvps: change.CauseExtra["wvps"].(string),
			TotalWvps: change.CauseExtra["total_wvps"].(string),
			Pool: pool,
		}).Error
	} else if change.Cause == "esys.reward.bp" {
		after := uint64(common.JsonNumber(change.Change.After.(json.Number)))
		before := uint64(common.JsonNumber(change.Change.Before.(json.Number)))
		return db.Create(&EcosysReward{
			BlockHeight: blockLog.BlockNum,
			BlockTime: time.Unix(int64(blockLog.BlockTime), 0),
			Name: change.Change.Id.(string),
			Type: change.Cause,
			Reward: after - before,
		}).Error
	}
	return nil
}

func (p *EcosysProcessor) ProcessOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error {
	return nil
}

func (p *EcosysProcessor) Finalize(db *gorm.DB, blockLog *blocklog.BlockLog) error {
	return nil
}
