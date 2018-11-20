package app

import (
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/db/storage"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"os"
	"testing"
)
const (
	dbPath = "./pbTool.db"
)
func Test_ApplyAccountCreate(t *testing.T) {
	clearDB()
	acop := &prototype.AccountCreateOperation{
		Fee:            prototype.MakeCoin(1),
		Creator:        &prototype.AccountName{Value: "initminer"},
		NewAccountName: &prototype.AccountName{Value: "alice"},
		Owner: &prototype.Authority{
			Cf:              prototype.Authority_owner,
			WeightThreshold: 1,
			AccountAuths: []*prototype.KvAccountAuth{
				&prototype.KvAccountAuth{
					Name:    &prototype.AccountName{Value: "initminer"},
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
		Active: &prototype.Authority{
		},
		Posting: &prototype.Authority{
		},
		MemoKey: &prototype.PublicKeyType{Data:[]byte{0}},
	}
	// construct base op ...
	op := &prototype.Operation{}
	op1 := &prototype.Operation_Op1{}
	op1.Op1 = acop
	op.Op = op1

	// init context

	db := startDB()
	defer db.Close()
	c := startController(db)

	ctx := &ApplyContext{ db:db, control:c}
	ev := &AccountCreateEvaluator{ctx:ctx,op:op.GetOp1()}
	ev.Apply()

	// verify
	name := &prototype.AccountName{Value:"alice"}
	accountWrap := table.NewSoAccountWrap(db,name)
	if !accountWrap.CheckExist() {
		t.Error("create new account failed ")
	}
}

func Test_ApplyTransfer(t *testing.T) {
	top := &prototype.TransferOperation{
		From: &prototype.AccountName{Value:"initminer"},
		To: &prototype.AccountName{Value:"alice"},
		Amount: prototype.MakeCoin(100),
	}

	db := startDB()
	defer db.Close()
	c := startController(db)

	alice := &prototype.AccountName{Value:"alice"}
	aliceWrap := table.NewSoAccountWrap(db,alice)
	aliceOrigin := aliceWrap.GetBalance().Value
	fmt.Println("alice origin:",aliceOrigin)

	initminer := &prototype.AccountName{Value:"initminer"}
	minerWrap := table.NewSoAccountWrap(db,initminer)
	initMinerOrigin := minerWrap.GetBalance().Value
	fmt.Println("initminer origin:",initMinerOrigin)

	// construct base op ...
	op := &prototype.Operation{}
	op2 := &prototype.Operation_Op2{}
	op2.Op2 = top
	op.Op = op2

	ctx := &ApplyContext{ db:db, control:c}
	ev := &TransferEvaluator{ctx:ctx,op:op.GetOp2()}
	ev.Apply()

	// check
	fmt.Println("alice new:",aliceWrap.GetBalance().Value)
	if aliceWrap.GetBalance().Value != aliceOrigin + 100 {
		t.Error("transfer op failed, receiver's balance wrong")
	}

	fmt.Println("initminer new:",minerWrap.GetBalance().Value)
	if minerWrap.GetBalance().Value != initMinerOrigin - 100 {
		t.Error("transfer op failed, sender's balance wrong")
	}
}

func startDB() iservices.IDatabaseService{
	db,err := storage.NewDatabase(dbPath)
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

func startController(db iservices.IDatabaseService) iservices.IController{
	c,_ := NewController(nil)
	c.SetDB(db)
	c.Open()
	return c
}

func clearDB() {
	os.RemoveAll(dbPath)
}