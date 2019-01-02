package vm

import (
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func TestVM_Estimate(t *testing.T) {
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

	applyOp := &prototype.ContractEstimateApplyOperation{
		Owner:    &prototype.AccountName{Value: "initminer"},
		Caller:   &prototype.AccountName{Value: "initminer"},
		Contract: "print",
		Params:   "",
	}
	signTx, err = dande.Sign(dande.InitminerPrivKey(), applyOp)
	myassert.Nil(err)
	dande.PushTrx(signTx)
	dande.GenerateBlock()
}
