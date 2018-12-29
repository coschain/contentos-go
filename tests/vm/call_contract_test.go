package vm

import (
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func TestVM_Deploy(t *testing.T) {
	myassert := assert.New(t)
	dande, _ := dandelion.NewGreenDandelion()
	_ = dande.OpenDatabase()
	defer func() {
		err := dande.Clean()
		if err != nil {
			t.Error(err)
		}
	}()
	db := dande.GetDB()
	data, _ := ioutil.ReadFile("./testdata/print.wasm")
	operation := &prototype.ContractDeployOperation{
		Owner:    &prototype.AccountName{Value: "initminer"},
		Contract: "print",
		Abi:      "",
		Code:     data,
	}
	signTx, err := dande.Sign(dande.InitminerPrivKey(), operation)
	myassert.Nil(err)
	dande.PushTrx(signTx)
	dande.GenerateBlock()

	cid := prototype.ContractId{Owner: &prototype.AccountName{Value: "initminer"}, Cname: "print"}
	scid := table.NewSoContractWrap(db, &cid)
	myassert.True(scid.CheckExist())
}

func TestVM_Call(t *testing.T) {
	myassert := assert.New(t)
	dande, _ := dandelion.NewGreenDandelion()
	_ = dande.OpenDatabase()
	defer func() {
		err := dande.Clean()
		if err != nil {
			t.Error(err)
		}
	}()
	data, _ := ioutil.ReadFile("./testdata/print.wasm")
	deployOp := &prototype.ContractDeployOperation{
		Owner:    &prototype.AccountName{Value: "initminer"},
		Contract: "print",
		Abi:      "",
		Code:     data,
	}
	signTx, err := dande.Sign(dande.InitminerPrivKey(), deployOp)
	myassert.Nil(err)
	dande.PushTrx(signTx)
	dande.GenerateBlock()

	db := dande.GetDB()

	cid := prototype.ContractId{Owner: &prototype.AccountName{Value: "initminer"}, Cname: "print"}
	scid := table.NewSoContractWrap(db, &cid)
	myassert.True(scid.CheckExist())

	err = dande.CreateAccount("kochiya")
	myassert.Nil(err)
	err = dande.Fund("kochiya", 5000)
	myassert.Nil(err)
	//db := dande.GetDB()
	acc := dande.GetAccount("kochiya")
	applyOp := &prototype.ContractApplyOperation{
		Caller:   &prototype.AccountName{Value: "kochiya"},
		Owner:    &prototype.AccountName{Value: "initminer"},
		Contract: "print",
		Amount:   &prototype.Coin{Value: 1000},
		Gas:      &prototype.Coin{Value: 2000},
	}
	signTx, err = dande.Sign(dande.GeneralPrivKey(), applyOp)
	myassert.Nil(err)
	dande.PushTrx(signTx)
	dande.GenerateBlock()

	acc = dande.GetAccount("kochiya")
	// transfer to contract 1000 gas
	// spent gas is unknown ---- even I stepped and got the value actually 541
	// the gas is enough, so the lower bound is 2000 and higher bound is 4000
	myassert.True(acc.GetBalance().Value < 4000)
	myassert.True(acc.GetBalance().Value > 2000)
	fmt.Println(acc.GetBalance())
}
