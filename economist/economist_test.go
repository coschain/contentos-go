package economist

import (
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"testing"
)

func TestEconomist_Mint(t *testing.T) {
	myassert := assert.New(t)
	dande, _ := dandelion.NewGreenDandelion()
	_ = dande.OpenDatabase()
	defer func() {
		err := dande.Clean()
		if err != nil {
			t.Error(err)
		}
	}()
	prop1 := dande.GetProps()
	myassert.Equal(prop1.PostRewards.Value, uint64(0))
	myassert.Equal(prop1.ReplyRewards.Value, uint64(0))
	var SingleId int32 = 1
	eco := Economist{dande.GetDB(), EventBus.New(), &SingleId, logrus.New()}
	bp, _ := eco.GetAccount(&prototype.AccountName{Value: "initminer"})
	eco.Mint()
	prop2 := dande.GetProps()
	myassert.Equal(prop2.PostRewards.Value, uint64(2397260))
	myassert.Equal(prop2.ReplyRewards.Value, uint64(479452))
	myassert.Equal(bp.GetVestingShares().Value, uint64(640270))
	eco.Mint()
	prop3 := dande.GetProps()
	myassert.Equal(prop3.PostRewards.Value, uint64(4794520))
	myassert.Equal(prop3.ReplyRewards.Value, uint64(958904))
	//keeper2, _ := eco.GetRewardsKeeper()
	//myassert.Equal(keeper2.Rewards["initminer"].Value, uint64(1278540))
	myassert.Equal(bp.GetVestingShares().Value, uint64(1279540))
}

func TestEconomist_Do(t *testing.T) {
	myassert := assert.New(t)
	dande, _ := dandelion.NewGreenDandelion()
	_ = dande.OpenDatabase()
	defer func() {
		err := dande.Clean()
		if err != nil {
			t.Error(err)
		}
	}()
	_ = dande.CreateAccount("kochiya")
	privKey := dande.GeneralPrivKey()
	db := dande.GetDB()
	var SingleId int32 = 1
	eco := Economist{dande.GetDB(), EventBus.New(),&SingleId, logrus.New()}
	operation := &prototype.PostOperation{
		Uuid:          uint64(111),
		Owner:         &prototype.AccountName{Value: "kochiya"},
		Title:         "Lorem Ipsum",
		Content:       "Lorem ipsum dolor sit amet",
		Tags:          []string{"article", "image"},
		Beneficiaries: []*prototype.BeneficiaryRouteType{},
	}
	signTx, err := dande.Sign(privKey, operation)
	myassert.Nil(err)
	dande.PushTrx(signTx)
	dande.GenerateBlocks(10)

	props := dande.GetProps()
	propsWrap := dande.GetPropsWrap()
	headTime := props.GetTime().UtcSeconds
	dande.GenerateBlock()
	uuid := uint64(111)
	postWrap := table.NewSoPostWrap(db, &uuid)
	//myassert.Equal(postWrap.GetTitle(), "Lorem Ipsum")
	postWrap.MdCashoutTime(&prototype.TimePointSec{UtcSeconds: headTime})
	postWrap.MdWeightedVp(100)
	props2 := dande.GetProps()
	props2.WeightedVps = 200
	props2.PostRewards = &prototype.Vest{Value: 1000}
	propsWrap.MdProps(props2)
	//fmt.Println(postWrap.GetCashoutTime())
	eco.Do()
	author, _ := eco.GetAccount(&prototype.AccountName{Value: "kochiya"})
	myassert.Equal(author.GetVestingShares().Value, uint64(334))

}
