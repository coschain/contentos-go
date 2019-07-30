package dandelion

import (
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/dandelion/core"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/vm/contract/abi"
	table2 "github.com/coschain/contentos-go/vm/contract/table"
	"github.com/sirupsen/logrus"
	"testing"
)

type Dandelion struct {
	*core.DandelionCore
}

func NewDandelion(logger *logrus.Logger) *Dandelion {
	return &Dandelion{
		DandelionCore: core.NewDandelionCore(logger),
	}
}

type DandelionTestFunc func(*testing.T, *Dandelion)

func NewDandelionTest(f DandelionTestFunc, actors int) func(*testing.T) {
	return func(t *testing.T) {
		d := NewDandelion(nil)
		if d == nil {
			t.Fatal("dandelion creation failed")
		}
		err := d.Start()
		if err != nil {
			t.Fatalf("dandelion start failed: %s", err.Error())
		}
		defer func() {
			_ = d.Stop()
		}()
		err = d.CreateAndFund("actor", actors, 100000 * constants.COSTokenDecimals, 10)
		if err != nil {
			t.Fatalf("dandelion createAndFund failed: %s", err.Error())
		}
		f(t, d)
	}
}

func (d *Dandelion) CreateAndFund(prefix string, n int, coins uint64, fee uint64) error {
	if n <= 0 {
		return nil
	}
	var ops []*prototype.Operation
	accounts := make(map[string]*prototype.PrivateKeyType)
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("%s%d", prefix, i)
		if d.Account(name).CheckExist() {
			return fmt.Errorf("account %s already exists", name)
		}
		priv, _ := prototype.GenerateNewKey()
		pub, _ := priv.PubKey()
		accounts[name] = priv
		ops = append(ops,
			AccountCreate(constants.COSInitMiner, name, pub, fee, ""),
			Transfer(constants.COSInitMiner, name, coins, ""))
	}
	if err := d.SendTrxByAccount(constants.COSInitMiner, ops...); err != nil {
		return err
	} else if err = d.ProduceBlocks(1); err != nil {
		return err
	}
	for name := range accounts {
		if !d.Account(name).CheckExist() {
			return fmt.Errorf("createAndFund account %s failed", name)
		}
	}
	for name, priv := range accounts {
		d.PutAccount(name, priv)
	}
	return nil
}

func (d *Dandelion) CreateAndFundUser(name string, coins uint64, fee uint64) error {
	if d.Account(name).CheckExist() {
		return fmt.Errorf("account %s already exists", name)
	}

	priv, _ := prototype.GenerateNewKey()
	pub, _ := priv.PubKey()
	err := d.SendTrxByAccount(constants.COSInitMiner,
		AccountCreate(constants.COSInitMiner, name, pub, fee, ""),
		Transfer(constants.COSInitMiner, name, coins, ""))

	if err != nil {
		return err
	} else if err = d.ProduceBlocks(1); err != nil {
		return err
	}

	if !d.Account(name).CheckExist() {
		return fmt.Errorf("createAndFund account %s failed", name)
	}
	return nil
}

func (d *Dandelion) Test(f DandelionTestFunc) func(*testing.T) {
	return func(t *testing.T) {
		f(t, d)
	}
}

//
// Table Record Retrieval by Primary keys
//

func (d *Dandelion) GlobalProps() *prototype.DynamicProperties {
	return d.TrxPool().GetProps()
}

func (d *Dandelion) Account(name string) *DandelionAccount {
	return NewDandelionAccount(name, d)
}

func (d *Dandelion) ExtTrx(trxId *prototype.Sha256) *table.SoExtTrxWrap {
	return table.NewSoExtTrxWrap(d.Database(), trxId)
}

func (d *Dandelion) ExtReward(account string, postId uint64) *table.SoExtRewardWrap {
	return table.NewSoExtRewardWrap(d.Database(), &prototype.RewardCashoutId{
		Account: prototype.NewAccountName(account),
		PostId: postId,
	})
}

func (d *Dandelion) ExtReplyCreated(postId uint64) *table.SoExtReplyCreatedWrap {
	return table.NewSoExtReplyCreatedWrap(d.Database(), &postId)
}

func (d *Dandelion) ExtUserPost(postId uint64) *table.SoExtUserPostWrap {
	return table.NewSoExtUserPostWrap(d.Database(), &postId)
}

func (d *Dandelion) Blocktrxs(block uint64) *table.SoBlocktrxsWrap {
	return table.NewSoBlocktrxsWrap(d.Database(), &block)
}

func (d *Dandelion) Vote(voter string, postId uint64) *table.SoVoteWrap {
	return table.NewSoVoteWrap(d.Database(), &prototype.VoterId{
		Voter: prototype.NewAccountName(voter),
		PostId: postId,
	})
}

func (d *Dandelion) ExtDailyTrx(date uint32) *table.SoExtDailyTrxWrap {
	return table.NewSoExtDailyTrxWrap(d.Database(), prototype.NewTimePointSec(date))
}

func (d *Dandelion) Post(postId uint64) *table.SoPostWrap {
	return table.NewSoPostWrap(d.Database(), &postId)
}

func (d *Dandelion) GiftTicket(ticket *prototype.GiftTicketKeyType) *table.SoGiftTicketWrap {
	return table.NewSoGiftTicketWrap(d.Database(), ticket)
}

func (d *Dandelion) Witness(owner string) *table.SoWitnessWrap {
	return table.NewSoWitnessWrap(d.Database(), prototype.NewAccountName(owner))
}

func (d *Dandelion) ExtFollowing(account string, following string) *table.SoExtFollowingWrap {
	return table.NewSoExtFollowingWrap(d.Database(), &prototype.FollowingRelation{
		Account: prototype.NewAccountName(account),
		Following: prototype.NewAccountName(following),
	})
}

func (d *Dandelion) TransactionObject(trxId *prototype.Sha256) *table.SoTransactionObjectWrap {
	return table.NewSoTransactionObjectWrap(d.Database(), trxId)
}

func (d *Dandelion) ReportList(uuid uint64) *table.SoReportListWrap {
	return table.NewSoReportListWrap(d.Database(), &uuid)
}

func (d *Dandelion) ExtFollowCount(account string) *table.SoExtFollowCountWrap {
	return table.NewSoExtFollowCountWrap(d.Database(), prototype.NewAccountName(account))
}

func (d *Dandelion) Contract(owner string, cname string) *table.SoContractWrap {
	return table.NewSoContractWrap(d.Database(), &prototype.ContractId{
		Owner: prototype.NewAccountName(owner),
		Cname: cname,
	})
}

func (d *Dandelion) ExtFollower(account string, follower string) *table.SoExtFollowerWrap {
	return table.NewSoExtFollowerWrap(d.Database(), &prototype.FollowerRelation{
		Account: prototype.NewAccountName(account),
		Follower: prototype.NewAccountName(follower),
	})
}

func (d *Dandelion) StakeRecord(from string, to string) *table.SoStakeRecordWrap {
	return table.NewSoStakeRecordWrap(d.Database(), &prototype.StakeRecord{
		From: prototype.NewAccountName(from),
		To: prototype.NewAccountName(to),
	})
}

func (d *Dandelion) BlockSummaryObject(id uint32) *table.SoBlockSummaryObjectWrap {
	return table.NewSoBlockSummaryObjectWrap(d.Database(), &id)
}

func (d *Dandelion) ExtHourTrx(hour uint32) *table.SoExtHourTrxWrap {
	return table.NewSoExtHourTrxWrap(d.Database(), prototype.NewTimePointSec(hour))
}

func (d *Dandelion) ExtPostCreated(postId uint64) *table.SoExtPostCreatedWrap {
	return table.NewSoExtPostCreatedWrap(d.Database(), &postId)
}

func (d *Dandelion) WitnessVote(voter string, witness string) *table.SoWitnessVoteWrap {
	return table.NewSoWitnessVoteWrap(d.Database(), &prototype.BpVoterId{
		Voter: prototype.NewAccountName(voter),
		Witness: prototype.NewAccountName(witness),
	})
}

//
// Contract tables
//
func (d *Dandelion) ContractTables(owner, contract string) *table2.ContractTables {
	if abiInterface, err := abi.UnmarshalABI([]byte(d.Contract(owner, contract).GetAbi())); err != nil {
		return nil
	} else {
		return table2.NewContractTables(owner, contract, abiInterface, d.Database())
	}
}
