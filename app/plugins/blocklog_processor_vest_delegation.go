package plugins

import (
	"fmt"
	"github.com/coschain/contentos-go/app/blocklog"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/jinzhu/gorm"
	"time"
)

const (
	VestDelegationCreated = "created"
	VestDelegationDelivering = "delivering"
	VestDelegationDone = "done"
)

type VestDelegationRecord struct {
	ID uint64					`gorm:"primary_key;auto_increment"`
	OrderID uint64				`gorm:"not null;unique_index"`
	From string					`gorm:"index"`
	To string					`gorm:"index"`
	Amount uint64				`gorm:"index"`
	CreatedAtBlock uint64		`gorm:"not null;index"`
	CreatedAtTime time.Time
	MaturedAtBlock uint64		`gorm:"not null;index"`
	ClaimedAtBlock uint64		`gorm:"index"`
	ClaimedAtTime time.Time
	DeliveredAtBlock uint64		`gorm:"index"`
	DeliveredAtTime time.Time
	Status string				`gorm:"index"`
}


type VestDelegationProcessor struct {
	tableReady bool
}

func NewVestDelegationProcessor() *VestDelegationProcessor {
	return &VestDelegationProcessor{}
}

func (p *VestDelegationProcessor) Prepare(db *gorm.DB, blockLog *blocklog.BlockLog) (err error) {
	if !p.tableReady {
		if !db.HasTable(&VestDelegationRecord{}) {
			if err = db.CreateTable(&VestDelegationRecord{}).Error; err == nil {
				p.tableReady = true
			}
		} else {
			p.tableReady = true
		}
	}
	return
}

func (p *VestDelegationProcessor) ProcessChange(db *gorm.DB, change *blocklog.StateChange, blockLog *blocklog.BlockLog, changeIdx, opIdx, trxIdx int) error {
	if change.What != "VestDelegation" {
		return nil
	}
	if change.Kind == blocklog.ChangeKindCreate && change.Cause == "delegate_vest" {
		rec := change.Change.After.(*table.SoVestDelegation)
		return db.Create(&VestDelegationRecord{
			OrderID:          rec.GetId(),
			From:             rec.GetFromAccount().GetValue(),
			To:               rec.GetToAccount().GetValue(),
			Amount:           rec.GetAmount().GetValue(),
			CreatedAtBlock:   rec.GetCreatedBlock(),
			CreatedAtTime:    time.Unix(int64(blockLog.BlockTime), 0),
			MaturedAtBlock:   rec.GetMaturityBlock(),
			ClaimedAtBlock:   0,
			ClaimedAtTime:    time.Unix(int64(constants.GenesisTime), 0),
			DeliveredAtBlock: 0,
			DeliveredAtTime:  time.Unix(int64(constants.GenesisTime), 0),
			Status:           VestDelegationCreated,
		}).Error
	} else if change.Kind == blocklog.ChangeKindUpdate && change.Cause == "un_delegate_vest" {
		rec := change.Change.After.(*table.SoVestDelegation)
		order := new(VestDelegationRecord)
		if db.Where(VestDelegationRecord{OrderID: rec.GetId()}).First(order).RecordNotFound() {
			return fmt.Errorf("vest delegation order not found. id=%d", rec.GetId())
		}
		order.ClaimedAtBlock = blockLog.BlockNum
		order.ClaimedAtTime = time.Unix(int64(blockLog.BlockTime), 0)
		order.DeliveredAtBlock = rec.GetDeliveryBlock()
		order.Status = VestDelegationDelivering
		return db.Save(order).Error
	} else if change.Kind == blocklog.ChangeKindDelete && change.Cause == "esys.deliver_vest" {
		rec := change.Change.Before.(*table.SoVestDelegation)
		order := new(VestDelegationRecord)
		if db.Where(VestDelegationRecord{OrderID: rec.GetId()}).First(order).RecordNotFound() {
			return fmt.Errorf("vest delegation order not found. id=%d", rec.GetId())
		}
		order.DeliveredAtTime = time.Unix(int64(blockLog.BlockTime), 0)
		order.Status = VestDelegationDone
		return db.Save(order).Error
	}
	return nil
}

func (p *VestDelegationProcessor) ProcessOperation(db *gorm.DB, blockLog *blocklog.BlockLog, opIdx, trxIdx int) error {
	return nil
}

func (p *VestDelegationProcessor) Finalize(db *gorm.DB, blockLog *blocklog.BlockLog) error {
	return nil
}


func init() {
	RegisterSQLTableNamePattern("vest_delegation_records")
}
