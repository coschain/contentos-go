package app

import (
	"testing"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/db/storage"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/app/table"
	"fmt"
)

func Test_ApplyAccountCreate(t *testing.T) {
	acop := &prototype.AccountCreateOperation{
		Fee:            &prototype.Coin{Amount: &prototype.Safe64{Value: 1}},
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

	fmt.Println("db:",db)
	fmt.Println("c:",c)

	ev := &AccountCreateEvaluator{}
	ev.SetDB(db)
	ev.SetController(c)
	ev.Apply(op)

	// verify
	name := &prototype.AccountName{Value:"alice"}
	accountWrap := table.NewSoAccountWrap(db,name)
	if !accountWrap.CheckExist() {
		t.Error("create new account failed ")
	}
}

func startDB() iservices.IDatabaseService{
	db,err := storage.NewDatabase("./pbTool.db")
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