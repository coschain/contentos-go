package plugins

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/prototype"
	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
)


type Op map[string]interface{}

func PurgeOperation(operations []*prototype.Operation) []Op {
	var ops []Op
	for _, operation := range operations {
		switch x := operation.Op.(type) {
		case *prototype.Operation_Op1:
			ops = append(ops, Op{"create_account":x.Op1})
		case *prototype.Operation_Op2:
			ops = append(ops, Op{"transfer": x.Op2})
		case *prototype.Operation_Op3:
			ops = append(ops, Op{"bp_register": x.Op3})
		case *prototype.Operation_Op4:
			ops = append(ops, Op{"bp_unregister": x.Op4})
		case *prototype.Operation_Op5:
			ops = append(ops, Op{"bp_vote": x.Op5})
		case *prototype.Operation_Op6:
			ops = append(ops, Op{"post": x.Op6})
		case *prototype.Operation_Op7:
			ops = append(ops, Op{"reply": x.Op7})
		case *prototype.Operation_Op8:
			ops = append(ops, Op{"follow": x.Op8})
		case *prototype.Operation_Op9:
			ops = append(ops, Op{"vote": x.Op9})
		case *prototype.Operation_Op10:
			ops = append(ops, Op{"transfer_to_vesting": x.Op10})
		case *prototype.Operation_Op13:
			ops = append(ops, Op{"contract_deploy": x.Op13})
		case *prototype.Operation_Op14:
			ops = append(ops, Op{"contract_apply": x.Op14})
		case *prototype.Operation_Op15:
			ops = append(ops, Op{"report": x.Op15})
		case *prototype.Operation_Op16:
			ops = append(ops, Op{"convert_vesting": x.Op16})
		}
	}
	return ops
}

var TrxMysqlServiceName = "trxmysql"

type TrxMysqlService struct {
	node.Service
	inDb  iservices.IDatabaseService
	outDb *sql.DB
	log *logrus.Logger
	ev  EventBus.Bus
	ctx *node.ServiceContext
}

func NewTrxMysqlSerVice(ctx *node.ServiceContext, log *logrus.Logger) (*TrxService, error) {
	return &TrxService{ctx: ctx, log: log}, nil
}

func (t *TrxMysqlService) Start(node *node.Node) error {
	inDb, err := t.ctx.Service(iservices.DbServerName)
	if err != nil {
		return err
	}
	t.inDb = inDb.(iservices.IDatabaseService)
	// dns to config file
	outDb, err := sql.Open("mysql", "contentos:123456@/contentosdb")
	if err != nil {
		return err
	}
	t.outDb = outDb
	t.ev = node.EvBus
	t.hookEvent()
	return nil
}

func (t *TrxMysqlService) hookEvent() {
	_ = t.ev.Subscribe(constants.NoticeLIB, t.handleLibNotification)
}
func (t *TrxMysqlService) unhookEvent() {
	_ = t.ev.Unsubscribe(constants.NoticeLIB, t.handleLibNotification)
}

func (t *TrxMysqlService) handleLibNotification(lib uint64) {
	sWrap := table.NewExtTrxBlockHeightWrap(t.inDb)
	start := lib
	end := lib + 1
	stmt, _ := t.outDb.Prepare("INSERT INTO trxinfo (trx_id, block_height, block_id, block_time, invoice, operations)  value (?, ?, ?, ?, ?, ?)")
	_ = sWrap.ForEachByOrder(&start, &end, nil, nil, func(trxKey *prototype.Sha256, blockHeight *uint64, idx uint32) bool {
		if trxKey != nil {
			wrap := table.NewSoExtTrxWrap(t.inDb, trxKey)
			if wrap != nil && wrap.CheckExist() {
				trxId := hex.EncodeToString(trxKey.GetHash())
				blockHeight := wrap.GetBlockHeight()
				blockId := hex.EncodeToString(wrap.GetBlockId().GetHash())
				blockTime := wrap.GetBlockTime().GetUtcSeconds()
				trxWrap := wrap.GetTrxWrap()
				invoice, _ := json.Marshal(trxWrap.Invoice)
				operations := PurgeOperation(trxWrap.SigTrx.GetTrx().GetOperations())
				operationsJson, _ := json.Marshal(operations)
				_, _ = stmt.Exec(trxId, blockHeight, blockId, blockTime, invoice, operationsJson)
				return true
			} else {
				return false
			}
		} else {
			return false
		}
	})
	defer stmt.Close()
}

func (t *TrxMysqlService) Stop() error {
	t.unhookEvent()
	_ = t.outDb.Close()
	return nil
}
