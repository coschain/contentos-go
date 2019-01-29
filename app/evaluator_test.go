package app

import (
	"fmt"
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/db/storage"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/mylog"
	"github.com/coschain/contentos-go/prototype"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

const (
	dbPath  = "./pbTool.db"
	logPath = "./logs"
)

var (
	SINGLE_ID int32 = 1
)

func tryCreateAccount(t *testing.T, acc string, ctrl *TrxPool) {
	acop := &prototype.AccountCreateOperation{
		Fee:            prototype.NewCoin(1),
		Creator:        &prototype.AccountName{Value: "initminer"},
		NewAccountName: &prototype.AccountName{Value: acc},
		Owner: &prototype.Authority{
			WeightThreshold: 1,
			AccountAuths: []*prototype.KvAccountAuth{
				&prototype.KvAccountAuth{
					Name:   &prototype.AccountName{Value: "initminer"},
					Weight: 3,
				},
			},
			KeyAuths: []*prototype.KvKeyAuth{
				&prototype.KvKeyAuth{
					Key: &prototype.PublicKeyType{
						Data: []byte{0},
					},
					Weight: 23,
				},
			},
		},
	}
	// construct base op ...
	op := &prototype.Operation{}
	op1 := &prototype.Operation_Op1{}
	op1.Op1 = acop
	op.Op = op1

	ctx := &ApplyContext{db: ctrl.db, control: ctrl}
	ev := &AccountCreateEvaluator{ctx: ctx, op: op.GetOp1()}
	ev.Apply()

	// verify
	name := &prototype.AccountName{Value: "alice"}
	accountWrap := table.NewSoAccountWrap(ctrl.db, name)
	if !accountWrap.CheckExist() {
		t.Error("create new account failed ")
	}
}

func Test_ApplyAccountCreate(t *testing.T) {
	// init context
	db := startDB()
	defer clearDB(db)

	c := startController(db)
	tryCreateAccount(t, "alice", c)
}

func Test_ApplyTransfer(t *testing.T) {
	top := &prototype.TransferOperation{
		From:   &prototype.AccountName{Value: "initminer"},
		To:     &prototype.AccountName{Value: "alice"},
		Amount: prototype.NewCoin(100),
	}

	db := startDB()
	defer clearDB(db)

	c := startController(db)

	tryCreateAccount(t, "alice", c)

	alice := &prototype.AccountName{Value: "alice"}
	aliceWrap := table.NewSoAccountWrap(db, alice)
	aliceOrigin := aliceWrap.GetBalance().Value
	fmt.Println("alice origin:", aliceOrigin)

	initminer := &prototype.AccountName{Value: "initminer"}
	minerWrap := table.NewSoAccountWrap(db, initminer)
	initMinerOrigin := minerWrap.GetBalance().Value
	fmt.Println("initminer origin:", initMinerOrigin)

	// construct base op ...
	op := &prototype.Operation{}
	op2 := &prototype.Operation_Op2{}
	op2.Op2 = top
	op.Op = op2

	ctx := &ApplyContext{db: db, control: c}
	ev := &TransferEvaluator{ctx: ctx, op: op.GetOp2()}
	ev.Apply()

	// check
	fmt.Println("alice new:", aliceWrap.GetBalance().Value)
	if aliceWrap.GetBalance().Value != aliceOrigin+100 {
		t.Error("transfer op failed, receiver's balance wrong")
	}

	fmt.Println("initminer new:", minerWrap.GetBalance().Value)
	if minerWrap.GetBalance().Value != initMinerOrigin-100 {
		t.Error("transfer op failed, sender's balance wrong")
	}
}

func TestPostEvaluator_ApplyNormal(t *testing.T) {
	operation := &prototype.PostOperation{
		Uuid:          uint64(111),
		Owner:         &prototype.AccountName{Value: "initminer"},
		Title:         "Lorem Ipsum",
		Content:       "Lorem ipsum dolor sit amet",
		Tags:          []string{"article", "image"},
		Beneficiaries: []*prototype.BeneficiaryRouteType{},
	}
	db := startDB()
	defer clearDB(db)

	op := &prototype.Operation{}
	opPost := &prototype.Operation_Op6{}
	opPost.Op6 = operation
	op.Op = opPost

	c := startController(db)

	props := c.GetProps()
	props.Time = &prototype.TimePointSec{UtcSeconds: uint32(time.Now().Unix())}
	c.updateGlobalDataToDB(props)
	ctx := &ApplyContext{db: db, control: c}
	ev := &PostEvaluator{ctx: ctx, op: op.GetOp6()}
	ev.Apply()

	uuid := uint64(111)
	postWrap := table.NewSoPostWrap(db, &uuid)
	myassert := assert.New(t)
	myassert.Equal(postWrap.GetAuthor().Value, "initminer")
	myassert.Equal(postWrap.GetPostId(), uint64(111))
	myassert.Equal(postWrap.GetRootId(), uint64(0))

	authorWrap := table.NewSoAccountWrap(ev.ctx.db, &prototype.AccountName{Value: "initminer"})
	// author last post time should be modified
	myassert.Equal(authorWrap.GetLastPostTime().UtcSeconds, uint32(time.Now().Unix()))
}

func TestPostEvaluator_ApplyPostExistId(t *testing.T) {
	operation := &prototype.PostOperation{
		Uuid:          uint64(111),
		Owner:         &prototype.AccountName{Value: "initminer"},
		Title:         "Lorem Ipsum",
		Content:       "Lorem ipsum dolor sit amet",
		Tags:          []string{"article", "image"},
		Beneficiaries: []*prototype.BeneficiaryRouteType{},
	}
	db := startDB()
	defer clearDB(db)

	op := &prototype.Operation{}
	opPost := &prototype.Operation_Op6{}
	opPost.Op6 = operation
	op.Op = opPost

	c := startController(db)

	props := c.GetProps()
	props.Time = &prototype.TimePointSec{UtcSeconds: uint32(time.Now().Unix())}
	c.updateGlobalDataToDB(props)

	ctx := &ApplyContext{db: db, control: c}
	ev := &PostEvaluator{ctx: ctx, op: op.GetOp6()}
	ev.Apply()

	// avoid frequently
	props.Time = &prototype.TimePointSec{UtcSeconds: uint32(time.Now().Unix() + 1000)}
	c.updateGlobalDataToDB(props)
	ctx = &ApplyContext{db: db, control: c}
	ev = &PostEvaluator{ctx: ctx, op: op.GetOp6()}
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("repost should have panic!")
			}
		}()

		ev.Apply()
	}()
}

func TestPostEvaluator_ApplyPostFrequently(t *testing.T) {
	operation := &prototype.PostOperation{
		Uuid:          uint64(111),
		Owner:         &prototype.AccountName{Value: "initminer"},
		Title:         "Lorem Ipsum",
		Content:       "Lorem ipsum dolor sit amet",
		Tags:          []string{"article", "image"},
		Beneficiaries: []*prototype.BeneficiaryRouteType{},
	}
	db := startDB()
	defer clearDB(db)

	op := &prototype.Operation{}
	opPost := &prototype.Operation_Op6{}
	opPost.Op6 = operation
	op.Op = opPost

	c := startController(db)

	props := c.GetProps()
	props.Time = &prototype.TimePointSec{UtcSeconds: uint32(time.Now().Unix())}
	c.updateGlobalDataToDB(props)

	ctx := &ApplyContext{db: db, control: c}
	ev := &PostEvaluator{ctx: ctx, op: op.GetOp6()}
	ev.Apply()

	operation = &prototype.PostOperation{
		Uuid:          uint64(112),
		Owner:         &prototype.AccountName{Value: "initminer"},
		Title:         "Lorem Ipsum",
		Content:       "Lorem ipsum dolor sit amet",
		Tags:          []string{"article", "image"},
		Beneficiaries: []*prototype.BeneficiaryRouteType{},
	}

	opPost.Op6 = operation
	op.Op = opPost

	ctx = &ApplyContext{db: db, control: c}
	ev = &PostEvaluator{ctx: ctx, op: op.GetOp6()}

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("repost should have panic!")
			}
		}()
		ev.Apply()
	}()
}

func TestReplyEvaluator_ApplyNormal(t *testing.T) {
	post_operation := &prototype.PostOperation{
		Uuid:          uint64(111),
		Owner:         &prototype.AccountName{Value: "initminer"},
		Title:         "Lorem Ipsum",
		Content:       "Lorem ipsum dolor sit amet",
		Tags:          []string{"article", "image"},
		Beneficiaries: []*prototype.BeneficiaryRouteType{},
	}
	db := startDB()
	defer clearDB(db)

	op := &prototype.Operation{}
	opPost := &prototype.Operation_Op6{}
	opPost.Op6 = post_operation
	op.Op = opPost

	c := startController(db)

	currentTimestamp := uint32(time.Now().Unix())

	props := c.GetProps()
	props.Time = &prototype.TimePointSec{UtcSeconds: currentTimestamp}
	c.updateGlobalDataToDB(props)
	ctx := &ApplyContext{db: db, control: c}
	ev := &PostEvaluator{ctx: ctx, op: op.GetOp6()}
	ev.Apply()

	props.Time = &prototype.TimePointSec{UtcSeconds: currentTimestamp + 1000}
	c.updateGlobalDataToDB(props)
	ctx = &ApplyContext{db: db, control: c}

	reply_operation := &prototype.ReplyOperation{
		Uuid:          uint64(112),
		Owner:         &prototype.AccountName{Value: "initminer"},
		Content:       "Lorem Ipsum",
		ParentUuid:    uint64(111),
		Beneficiaries: []*prototype.BeneficiaryRouteType{},
	}

	op = &prototype.Operation{}
	opReply := &prototype.Operation_Op7{}
	opReply.Op7 = reply_operation
	op.Op = opReply

	ev2 := &ReplyEvaluator{ctx: ctx, op: op.GetOp7()}
	ev2.Apply()

	uuid := uint64(112)
	postWrap := table.NewSoPostWrap(db, &uuid)
	myassert := assert.New(t)
	myassert.Equal(postWrap.GetAuthor().Value, "initminer")
	myassert.Equal(postWrap.GetPostId(), uint64(112))
	myassert.Equal(postWrap.GetRootId(), uint64(111))
	myassert.Equal(postWrap.GetParentId(), uint64(111))

	authorWrap := table.NewSoAccountWrap(ev.ctx.db, &prototype.AccountName{Value: "initminer"})
	myassert.Equal(authorWrap.GetLastPostTime().UtcSeconds, currentTimestamp+1000)
}

func TestVoteEvaluator_ApplyNormal(t *testing.T) {
	post_operation := &prototype.PostOperation{
		Uuid:          uint64(111),
		Owner:         &prototype.AccountName{Value: "initminer"},
		Title:         "Lorem Ipsum",
		Content:       "Lorem ipsum dolor sit amet",
		Tags:          []string{"article", "image"},
		Beneficiaries: []*prototype.BeneficiaryRouteType{},
	}
	db := startDB()
	defer clearDB(db)

	op := &prototype.Operation{}
	opPost := &prototype.Operation_Op6{}
	opPost.Op6 = post_operation
	op.Op = opPost

	c := startController(db)

	currentTimestamp := c.HeadBlockTime().UtcSeconds + 1000

	props := c.GetProps()
	//props.Time = &prototype.TimePointSec{UtcSeconds: currentTimestamp}
	props.Time = &prototype.TimePointSec{UtcSeconds: currentTimestamp}
	c.updateGlobalDataToDB(props)
	ctx := &ApplyContext{db: db, control: c}
	ev := &PostEvaluator{ctx: ctx, op: op.GetOp6()}
	ev.Apply()

	vote_operation := &prototype.VoteOperation{
		Voter: &prototype.AccountName{Value: "initminer"},
		Idx:   uint64(111),
	}

	op = &prototype.Operation{}
	opVote := &prototype.Operation_Op9{}
	opVote.Op9 = vote_operation
	op.Op = opVote

	ev2 := &VoteEvaluator{ctx: ctx, op: op.GetOp9()}
	ev2.Apply()

	myassert := assert.New(t)

	uuid := uint64(111)
	postWrap := table.NewSoPostWrap(db, &uuid)

	voterWrap := table.NewSoAccountWrap(ev.ctx.db, &prototype.AccountName{Value: "initminer"})
	myassert.Equal(voterWrap.GetVotePower(), uint32(36))
	myassert.Equal(voterWrap.GetLastPostTime().UtcSeconds, c.HeadBlockTime().UtcSeconds)
	myassert.Equal(postWrap.GetWeightedVp(), uint64(2000))
	myassert.Equal(c.GetProps().WeightedVps, uint64(2000))

}

func startDB() iservices.IDatabaseService {
	dir, _ := ioutil.TempDir("", "db")
	db, err := storage.NewDatabase(filepath.Join(dir, strconv.FormatUint(rand.Uint64(), 16)))
	if err != nil {
		return nil
	}
	err = db.Start(nil)
	if err != nil {
		fmt.Print(err)
		panic("start db error")
	}
	return db
}

func clearDB(db iservices.IDatabaseService) {
	db.Close()
	dir, _ := ioutil.TempDir("", "db")
	os.RemoveAll(dir)
}

func startController(db iservices.IDatabaseService) *TrxPool {
	log, err := mylog.NewMyLog(logPath, mylog.DebugLevel, 0)
	mustNoError(err, "new log error")
	c, _ := NewController(nil, log.Logger)
	c.SetDB(db)
	c.SetBus(EventBus.New())
	c.Open()
	c.SetShuffle(func(block common.ISignedBlock) {
	})
	return c
}
