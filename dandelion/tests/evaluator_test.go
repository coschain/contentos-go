package tests

import (
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	dande "github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	SINGLE_ID int32 = 1
)

func TestPostEvaluator_DandelionNormal(t *testing.T) {
	myassert := assert.New(t)
	dandelion, _ := dande.NewGreenDandelion()
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

func TestClaimEvaluator_DandelionNormal(t *testing.T) {
	myassert := assert.New(t)
	dandelion, _ := dande.NewGreenDandelion()
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
	dandelion.GenerateBlocks(1000)
	keeperWrap := table.NewSoRewardsKeeperWrap(db, &SINGLE_ID)
	keeper := keeperWrap.GetKeeper()
	fmt.Println(keeper.Rewards["kochiya"])
	acc := dandelion.GetAccount("kochiya")

	fmt.Println(acc.GetBalance())

	//claimOperation := &prototype.ClaimOperation{
	//	Account:
	//}

}
