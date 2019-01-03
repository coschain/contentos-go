package tests

import (
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

	keeperWrapper := table.NewSoRewardsKeeperWrap(db, &SINGLE_ID)
	keeper := keeperWrapper.GetKeeper()
	keeper.Rewards["kochiya"] = &prototype.Vest{Value: 1000}
	keeperWrapper.MdKeeper(keeper)
	operation := &prototype.ClaimOperation{
		Account: &prototype.AccountName{Value: "kochiya"},
		Amount:  500,
	}
	signTx, err := dandelion.Sign(privKey, operation)
	myassert.Nil(err)
	dandelion.PushTrx(signTx)
	dandelion.GenerateBlock()

	acc := table.NewSoAccountWrap(db, &prototype.AccountName{Value: "kochiya"})
	myassert.Equal(acc.GetVestingShares().Value, uint64(501))
	keeperWrapper = table.NewSoRewardsKeeperWrap(db, &SINGLE_ID)
	keeper = keeperWrapper.GetKeeper()
	reward := keeper.Rewards
	myassert.Equal(reward["kochiya"].Value, uint64(500))
}

func TestClaimEvaluator_DandelionOvercharge(t *testing.T) {
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

	keeperWrapper := table.NewSoRewardsKeeperWrap(db, &SINGLE_ID)
	keeper := keeperWrapper.GetKeeper()
	keeper.Rewards["kochiya"] = &prototype.Vest{Value: 1000}
	keeperWrapper.MdKeeper(keeper)
	operation := &prototype.ClaimOperation{
		Account: &prototype.AccountName{Value: "kochiya"},
		Amount:  1001,
	}
	signTx, err := dandelion.Sign(privKey, operation)
	myassert.Nil(err)
	dandelion.PushTrx(signTx)
	dandelion.GenerateBlock()

	acc := table.NewSoAccountWrap(db, &prototype.AccountName{Value: "kochiya"})
	myassert.Equal(acc.GetVestingShares().Value, uint64(1001))
	keeperWrapper = table.NewSoRewardsKeeperWrap(db, &SINGLE_ID)
	keeper = keeperWrapper.GetKeeper()
	reward := keeper.Rewards
	myassert.Equal(reward["kochiya"].Value, uint64(0))
}

func TestClaimallEvaluator_DandelionNormal(t *testing.T) {
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

	keeperWrapper := table.NewSoRewardsKeeperWrap(db, &SINGLE_ID)
	keeper := keeperWrapper.GetKeeper()
	keeper.Rewards["kochiya"] = &prototype.Vest{Value: 1000}
	keeperWrapper.MdKeeper(keeper)
	operation := &prototype.ClaimAllOperation{
		Account: &prototype.AccountName{Value: "kochiya"},
	}
	signTx, err := dandelion.Sign(privKey, operation)
	myassert.Nil(err)
	dandelion.PushTrx(signTx)
	dandelion.GenerateBlock()

	acc := table.NewSoAccountWrap(db, &prototype.AccountName{Value: "kochiya"})
	myassert.Equal(acc.GetVestingShares().Value, uint64(1001))
	keeperWrapper = table.NewSoRewardsKeeperWrap(db, &SINGLE_ID)
	keeper = keeperWrapper.GetKeeper()
	reward := keeper.Rewards
	myassert.Equal(reward["kochiya"].Value, uint64(0))
}
