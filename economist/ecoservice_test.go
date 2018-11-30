package economist

import (
	"fmt"
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/app"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/db/storage"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

const (
	dbPath = "./pbTool.db"
)

func startDB() iservices.IDatabaseService {
	db, err := storage.NewDatabase(dbPath)
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

func startController(db iservices.IDatabaseService) *app.Controller {
	c, _ := app.NewController(nil)
	c.SetDB(db)
	c.SetBus(EventBus.New())
	c.Open()
	return c
}

func clearDB() {
	_ = os.RemoveAll(dbPath)
}

func TestEconomist_Do(t *testing.T) {
	clearDB()
	db := startDB()
	defer db.Close()
	myassert := assert.New(t)

	c := startController(db)

	dgpWrap := table.NewSoGlobalWrap(db, &SINGLE_ID)
	if !dgpWrap.CheckExist() {
		t.Error("dgpwrap check exist error")
	}
	globalProps := dgpWrap.GetProps()
	keeperWrap := table.NewSoRewardsKeeperWrap(db, &SINGLE_ID)
	if !keeperWrap.CheckExist() {
		t.Error("keep wrap check exist error")
	}
	rewardsKeeper := keeperWrap.GetKeeper()
	e := &Economist{ctx: nil, db: db, rewardAccumulator: 0, vpAccumulator: 0, globalProps: globalProps,
		rewardsKeeper: rewardsKeeper}

	// post an article
	post_operation := &prototype.PostOperation{
		Uuid:          uint64(111),
		Owner:         &prototype.AccountName{Value: "initminer"},
		Title:         "Lorem Ipsum",
		Content:       "Lorem ipsum dolor sit amet",
		Tags:          []string{"article", "image"},
		Beneficiaries: []*prototype.BeneficiaryRouteType{},
	}

	op := &prototype.Operation{}
	opPost := &prototype.Operation_Op6{}
	opPost.Op6 = post_operation
	op.Op = opPost

	// mock has
	globalProps.WeightedVps = 1000
	globalProps.PostRewards = &prototype.Vest{Value: 1000}

	currentTimestamp := c.HeadBlockTime().UtcSeconds + 1000

	globalProps.Time = &prototype.TimePointSec{UtcSeconds: currentTimestamp}
	dgpWrap.MdProps(globalProps)

	postWrap := table.NewSoPostWrap(db, &post_operation.Uuid)
	err := postWrap.Create(func(t *table.SoPost) {
		t.PostId = post_operation.Uuid
		t.Tags = post_operation.Tags
		t.Title = post_operation.Title
		t.Author = post_operation.Owner
		t.Body = post_operation.Content
		t.Created = c.HeadBlockTime()
		t.CashoutTime = &prototype.TimePointSec{UtcSeconds: c.HeadBlockTime().UtcSeconds + uint32(constants.POST_CASHPUT_DELAY_TIME)}
		t.Depth = 0
		t.Children = 0
		t.RootId = t.PostId
		t.ParentId = 0
		t.RootId = 0
		t.Beneficiaries = post_operation.Beneficiaries
		t.WeightedVp = 1000
		t.VoteCnt = 0
	})

	myassert.NoError(err, "create post success")

	timestamp := postWrap.GetCashoutTime().UtcSeconds - uint32(constants.GenesisTime)
	key := fmt.Sprintf("cashout:%d_%d", common.GetBucket(timestamp), post_operation.Uuid)
	value := "post"
	err = db.Put([]byte(key), []byte(value))
	myassert.NoError(err, "put into db error")

	// jump to cashout time
	globalProps.Time = &prototype.TimePointSec{UtcSeconds: c.HeadBlockTime().UtcSeconds + uint32(constants.POST_CASHPUT_DELAY_TIME)}
	dgpWrap.MdProps(globalProps)

	err = e.Do()

	keeper := keeperWrap.GetKeeper()
	myassert.Equal(keeper.Rewards["initminer"].Value, uint64(1000))
}
