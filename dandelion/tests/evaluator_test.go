package tests

import (
	"github.com/coschain/contentos-go/app/table"
	dande "github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPostEvaluator_DandelionNormal(t *testing.T) {
	myassert := assert.New(t)
	dandelion, _ := dande.NewDandelion()
	_ = dandelion.OpenDatabase()
	defer func() {
		err := dandelion.Clean()
		if err != nil {
			t.Error(err)
		}
	}()
	_ = dandelion.CreateAccount("kochiya")
	privKey := dandelion.GeneralPrivKey()
	db := dandelion.GetDB()
	operation := &prototype.PostOperation{
		Uuid:          uint64(111),
		Owner:         &prototype.AccountName{Value: "kochiya"},
		Title:         "Lorem Ipsum",
		Content:       "Lorem ipsum dolor sit amet",
		Tags:          []string{"article", "image"},
		Beneficiaries: []*prototype.BeneficiaryRouteType{},
	}
	signTx, err := dandelion.Sign(privKey, operation)
	myassert.Nil(err)
	dandelion.PushTrx(signTx)
	dandelion.GenerateBlock()

	uuid := uint64(111)
	postWrap := table.NewSoPostWrap(db, &uuid)
	myassert.Equal(postWrap.GetTitle(), "Lorem Ipsum")
}
