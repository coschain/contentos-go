package op

import (
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common/constants"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

type VestDelegationTester struct {
	acc0, acc1, acc2, acc3, acc4  *DandelionAccount
	bpNum, threshold int
	bpList []*DandelionAccount
}

func (tester *VestDelegationTester) TestBaseFunction1(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")
	tester.acc3 = d.Account("actor3")
	tester.acc4 = d.Account("actor4")


	// vest delegation only works after hard fork 3.
	t.Run("hard_fork3_switch", d.Test(tester.hardFork3Switch))

	// TODO: missing test cases (maybe incomplete)
	// 1, invalid delegate operation params (op.amount or op.expiration out of range) -> refused by producers
	// 2, insufficient vest -> status: 201
	// 3, delegation amount overflow test
	// 4, normal delegation
	// 5, early undelegation (undelegate before order maturity)
	// 6, normal undelegation
	// 7, delivery test (order should be removed after delivery)
	// 8, bp votes checking when account's vest changes
	// 9, related tests: cross-affection between power-down and vest delegation

	t.Run("base function", d.Test(tester.baseFunction1))
}

func (tester *VestDelegationTester) TestBaseFunction2(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	t.Run("base function", d.Test(tester.baseFunction2))
}

func (tester *VestDelegationTester) TestBpRelated(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")
	tester.acc3 = d.Account("actor3")
	tester.acc4 = d.Account("actor4")

	t.Run("test bp related", d.Test(tester.bpRelated))
}

func (tester *VestDelegationTester) TestPowerDownRelated(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")

	t.Run("test power down related", d.Test(tester.powerDownRelated))
}

func (tester *VestDelegationTester) TestMultiDelegation(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")
	tester.acc3 = d.Account("actor3")

	t.Run("test multi delegation", d.Test(tester.multiDelegation))
}

func (tester *VestDelegationTester) hardFork3Switch(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	opDelegation := DelegateVest(tester.acc0.Name, tester.acc1.Name, constants.MinVestDelegationAmount, constants.MinVestDelegationInBlocks)
	opUnDelegation := UnDelegateVest(tester.acc0.Name, 1)

	//
	// before hard fork 3, delegation & un_delegation transactions should be ignored by block producers.
	// transactions can be signed, but blocks never contain delegation & un_delegation transactions.
	//

	// hard fork 0
	tester.shouldRefuseTransaction(true, t, tester.acc0, opDelegation)
	tester.shouldRefuseTransaction(true, t, tester.acc0, opUnDelegation)

	// hard fork 1
	a.NoError(d.ProduceBlocks(int(constants.HardFork1) - int(d.GlobalProps().HeadBlockNumber)))
	tester.shouldRefuseTransaction(true, t, tester.acc0, opDelegation)
	tester.shouldRefuseTransaction(true, t, tester.acc0, opUnDelegation)

	// hard fork 2
	a.NoError(d.ProduceBlocks(int(constants.HardFork2) - int(d.GlobalProps().HeadBlockNumber)))
	tester.shouldRefuseTransaction(true, t, tester.acc0, opDelegation)
	tester.shouldRefuseTransaction(true, t, tester.acc0, opUnDelegation)

	// hard fork 3
	a.NoError(d.ProduceBlocks(int(constants.HardFork3) - int(d.GlobalProps().HeadBlockNumber)))
	tester.shouldRefuseTransaction(false, t, tester.acc0, opDelegation)
	tester.shouldRefuseTransaction(false, t, tester.acc0, opUnDelegation)
}

func (tester *VestDelegationTester) baseFunction1(t *testing.T, d *Dandelion) {
	// actor0 has 10000.1 vest, actor1 has 0.1 vest, actor2 has 0.1 vest
	a := assert.New(t)
	a.NoError(d.ProduceBlocks( int(constants.HardFork3) + 1 ))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(TransferToVest(tester.acc0.Name, tester.acc0.Name, 10000000000, "")))

	// actor0 delegation expiration too short -> failed
	expiration := uint64(0)
	opDelegation := DelegateVest(tester.acc0.Name, tester.acc1.Name, constants.MinVestDelegationAmount, expiration)
	a.Error(tester.acc0.SendTrxAndProduceBlock(opDelegation))

	// actor0 delegation expiration too long -> failed
	expiration = constants.MaxVestDelegationInBlocks + 1
	opDelegation = DelegateVest(tester.acc0.Name, tester.acc1.Name, constants.MinVestDelegationAmount, expiration)
	a.Error(tester.acc0.SendTrxAndProduceBlock(opDelegation))

	// actor0 delegate 0 vest to actor1 (amount too small) -> failed
	delegateAmount := uint64(0)
	opDelegation = DelegateVest(tester.acc0.Name, tester.acc1.Name, delegateAmount, constants.MinVestDelegationInBlocks)
	a.Error(tester.acc0.SendTrxAndProduceBlock(opDelegation))

	// actor0 delegate 20000 vest to actor1 (insufficient vest) -> failed
	delegateAmount = uint64(20000000000)
	opDelegation = DelegateVest(tester.acc0.Name, tester.acc1.Name, delegateAmount, constants.MinVestDelegationInBlocks)
	a.Error(tester.acc0.SendTrxAndProduceBlock(opDelegation))

	// actor0 delegate maxUint64 + 1 vest to actor1 -> failed
	opDelegation = DelegateVest(tester.acc0.Name, tester.acc1.Name, math.MaxUint64, constants.MinVestDelegationInBlocks)
	a.Error(tester.acc0.SendTrxAndProduceBlock(opDelegation))

	// actor0 delegate 10000.1 vest to actor1 -> failed
	delegateAmount = uint64(10000100000)
	opDelegation = DelegateVest(tester.acc0.Name, tester.acc1.Name, delegateAmount, constants.MinVestDelegationInBlocks)
	a.Error(tester.acc0.SendTrxAndProduceBlock(opDelegation))

	// actor0 delegate to himself -> failed
	delegateAmount = constants.MinVestDelegationAmount
	opDelegation = DelegateVest(tester.acc0.Name, tester.acc0.Name, delegateAmount, constants.MinVestDelegationInBlocks)
	a.Error(tester.acc0.SendTrxAndProduceBlock(opDelegation))

	// set actor0's reputation to MinReputation
	a.NoError(DeploySystemContract(constants.COSSysAccount, repCrtName, d))
	a.True(d.Contract(constants.COSSysAccount, repCrtName).CheckExist())
	bpList := []*DandelionAccount{tester.acc0, tester.acc1, tester.acc2, tester.acc3}
	if a.NoError(RegisterBp(bpList, d)) {
		tester.bpNum = len(bpList)
		tester.threshold = tester.bpNum/3*2 + 1
		tester.bpList = append(tester.bpList, bpList...)
	}
	d.ProduceBlocks(SysCtrPropExpTime + 1)
	adminName := tester.acc4.Name
	ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.proposal %q", tester.acc0.Name, constants.COSSysAccount, repCrtName, adminName))
	for i := 0; i < tester.threshold; i++ {
		name := fmt.Sprintf("actor%d", i)
		ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.vote ", name, constants.COSSysAccount, repCrtName))
	}
	d.ProduceBlocks(SysCtrPropExpTime + 1)
	admin := d.GlobalProps().ReputationAdmin
	a.NotNil(admin)
	mdRep := uint32(constants.MinReputation)
	newMemo := GetNewMemo(tester.acc0.GetReputationMemo())
	name0 := tester.acc0.Name
	ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.setrep %q, %d, %q", admin.Value, constants.COSSysAccount, repCrtName, name0, mdRep, newMemo))

	// actor0 delegate 10000 vest to actor1 but his reputation is 0  -> failed
	delegateAmount = uint64(10000000000)
	opDelegation = DelegateVest(tester.acc0.Name, tester.acc1.Name, delegateAmount, constants.MinVestDelegationInBlocks)
	a.Error(tester.acc0.SendTrxAndProduceBlock(opDelegation))
}

func (tester *VestDelegationTester) baseFunction2(t *testing.T, d *Dandelion) {
	// actor0 has 10000.1 vest, actor1 has 0.1 vest, actor2 has 0.1 vest
	a := assert.New(t)
	a.NoError(d.ProduceBlocks( int(constants.HardFork3) + 1 ))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(TransferToVest(tester.acc0.Name, tester.acc0.Name, 10000000000, "")))

	// create a block producer to start the economy system
	name := "testacc"
	makeBp(name,t,d)
	witWrap := d.BlockProducer(name)
	a.True(witWrap.CheckExist())

	// actor0 delegate 10000 vest to actor1 and has enough reputation -> success
	// actor0's vest decrease 10000 and actor1's vest increase 10000 immediately
	delegateAmount := uint64(10000000000)
	oldVestAcc0 := tester.acc0.GetVest()
	oldVestAcc1 := tester.acc1.GetVest()
	opDelegation := DelegateVest(tester.acc0.Name, tester.acc1.Name, delegateAmount, constants.MinVestDelegationInBlocks)
	a.NoError(tester.acc0.SendTrxAndProduceBlock(opDelegation))
	newVestAcc0 := tester.acc0.GetVest()
	newVestAcc1 := tester.acc1.GetVest()
	a.Equal(oldVestAcc0.Value - newVestAcc0.Value, delegateAmount)
	a.Equal(newVestAcc1.Value - oldVestAcc1.Value, delegateAmount)

	// actor1's borrowed vest is 10000
	a.Equal(tester.acc1.GetBorrowedVest().Value, delegateAmount)

	// actor1 delegate 10000 vest to actor2 -> failed
	opDelegation = DelegateVest(tester.acc1.Name, tester.acc2.Name, delegateAmount, constants.MinVestDelegationInBlocks)
	a.Error(tester.acc0.SendTrxAndProduceBlock(opDelegation))

	// before maturity actor0 undelegate -> failed
	opUnDelegation := UnDelegateVest(tester.acc0.Name, 1)
	a.Error(tester.acc0.SendTrxAndProduceBlock(opUnDelegation))

	a.NoError(d.ProduceBlocks( int(constants.MinVestDelegationInBlocks) + 1 ))

	// after maturity actor0 undelegate 10000 vest -> success
	// actor1's vest decrease 10000 immediately and actor0's vest not change
	oldVestAcc0 = tester.acc0.GetVest()
	oldVestAcc1 = tester.acc1.GetVest()
	opUnDelegation = UnDelegateVest(tester.acc0.Name, 1)
	a.NoError(tester.acc0.SendTrxAndProduceBlock(opUnDelegation))
	newVestAcc0 = tester.acc0.GetVest()
	newVestAcc1 = tester.acc1.GetVest()
	a.Equal(oldVestAcc0.Value,  newVestAcc0.Value)
	a.Equal(oldVestAcc1.Value - newVestAcc1.Value, delegateAmount)

	// actor0's delivery vest is 10000 vest
	a.Equal(tester.acc0.GetDeliveringVest().Value, delegateAmount)
	// actor1's borrowed vest is 0
	a.Equal(tester.acc1.GetBorrowedVest().Value, uint64(0))

	// duplicate undelegate
	opUnDelegation = UnDelegateVest(tester.acc0.Name, 1)
	a.Error(tester.acc0.SendTrxAndProduceBlock(opUnDelegation))

	// no exist undelegate
	opUnDelegation = UnDelegateVest(tester.acc0.Name, 100)
	a.Error(tester.acc0.SendTrxAndProduceBlock(opUnDelegation))

	// mismatch undelegate
	opUnDelegation = UnDelegateVest(tester.acc1.Name, 1)
	a.Error(tester.acc1.SendTrxAndProduceBlock(opUnDelegation))

	// before VestDelegationDeliveryInBlocks actor0 delegate 10000 vest -> failed
	opDelegation = DelegateVest(tester.acc0.Name, tester.acc1.Name, delegateAmount, constants.MinVestDelegationInBlocks)
	a.Error(tester.acc0.SendTrxAndProduceBlock(opDelegation))

	// after VestDelegationDeliveryInBlocks actor0's vest increase 10000
	a.NoError(d.ProduceBlocks( int(constants.VestDelegationDeliveryInBlocks) + 1 ))
	newVestAcc0 = tester.acc0.GetVest()
	a.Equal(newVestAcc0.Value - oldVestAcc0.Value, delegateAmount)

	// delegate record should be deleted
	orderId := uint64(1)
	delegateWrap := table.NewSoVestDelegationWrap(d.Database(), &orderId)
	a.False(delegateWrap.CheckExist())

	// actor0's delivery vest is 0 vest
	a.Equal(tester.acc0.GetDeliveringVest().Value, uint64(0))

	// after maturity actor0 undelegate and use the budget id which has already been undelegated -> failed
	opUnDelegation = UnDelegateVest(tester.acc0.Name, 1)
	a.Error(tester.acc0.SendTrxAndProduceBlock(opUnDelegation))
}

func (tester *VestDelegationTester) bpRelated(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.NoError(d.ProduceBlocks( int(constants.HardFork3) + 1 ))

	// actor0 has 10000.1 vest， actor1 has 0.1 vest
	// actor3 and actor4 are producers, actor0 vote actor3, actor1 vote actor4
	a.NoError(tester.acc0.SendTrxAndProduceBlock(TransferToVest(tester.acc0.Name, tester.acc0.Name, 10000000000, "")))

	bpList := []*DandelionAccount{tester.acc3, tester.acc4}
	a.NoError(RegisterBp(bpList, d))
	opBpVote := BpVote(tester.acc0.Name, tester.acc3.Name, false)
	a.NoError(tester.acc0.SendTrxAndProduceBlock(opBpVote))
	opBpVote = BpVote(tester.acc1.Name, tester.acc4.Name, false)
	a.NoError(tester.acc1.SendTrxAndProduceBlock(opBpVote))
	wit3Wrap := d.BlockProducer(tester.acc3.Name)
	a.True(wit3Wrap.CheckExist())
	wit4Wrap := d.BlockProducer(tester.acc4.Name)
	a.True(wit4Wrap.CheckExist())

	// actor0 delegate 10000 vest to actor1 and has enough reputation -> success
	// vote count of actor3 decrease 10000 and vote count of actor4 increase 10000 immediately
	delegateAmount := uint64(10000000000)
	oldVoteVestAcc3 := wit3Wrap.GetBpVest().VoteVest
	oldVoteVestAcc4 := wit4Wrap.GetBpVest().VoteVest
	opDelegation := DelegateVest(tester.acc0.Name, tester.acc1.Name, delegateAmount, constants.MinVestDelegationInBlocks)
	a.NoError(tester.acc0.SendTrxAndProduceBlock(opDelegation))
	newVoteVestAcc3 := wit3Wrap.GetBpVest().VoteVest
	newVoteVestAcc4 := wit4Wrap.GetBpVest().VoteVest
	a.Equal(oldVoteVestAcc3.Value - newVoteVestAcc3.Value, delegateAmount)
	a.Equal(newVoteVestAcc4.Value - oldVoteVestAcc4.Value, delegateAmount)

	// after maturity actor0 undelegate 10000 vest -> success
	// vote count of actor4 decrease 10000 immediately and vote count of actor3 not change
	a.NoError(d.ProduceBlocks( int(constants.MinVestDelegationInBlocks) + 1 ))
	oldVoteVestAcc3 = wit3Wrap.GetBpVest().VoteVest
	oldVoteVestAcc4 = wit4Wrap.GetBpVest().VoteVest
	opUnDelegation := UnDelegateVest(tester.acc0.Name, 1)
	a.NoError(tester.acc0.SendTrxAndProduceBlock(opUnDelegation))
	newVoteVestAcc3 = wit3Wrap.GetBpVest().VoteVest
	newVoteVestAcc4 = wit4Wrap.GetBpVest().VoteVest
	a.Equal(oldVoteVestAcc3.Value, newVoteVestAcc3.Value)
	a.Equal(oldVoteVestAcc4.Value - newVoteVestAcc4.Value, delegateAmount)


	// before VestDelegationDeliveryInBlocks actor1 vote actor3, vote count of actor3 increase only 0.1 vest
	oldVoteVestAcc3 = wit3Wrap.GetBpVest().VoteVest
	opBpVote = BpVote(tester.acc1.Name, tester.acc4.Name, true)
	a.NoError(tester.acc1.SendTrxAndProduceBlock(opBpVote))
	opBpVote = BpVote(tester.acc1.Name, tester.acc3.Name, false)
	a.NoError(tester.acc1.SendTrxAndProduceBlock(opBpVote))
	newVoteVestAcc3 = wit3Wrap.GetBpVest().VoteVest
	a.Equal(oldVoteVestAcc3.Value + constants.DefaultAccountCreateFee, newVoteVestAcc3.Value)

	// after VestDelegationDeliveryInBlocks vote count of actor3 increase 10000
	oldVoteVestAcc3 = wit3Wrap.GetBpVest().VoteVest
	a.NoError(d.ProduceBlocks( int(constants.VestDelegationDeliveryInBlocks) + 1 ))
	newVoteVestAcc3 = wit3Wrap.GetBpVest().VoteVest
	a.Equal(newVoteVestAcc3.Value - oldVoteVestAcc3.Value, delegateAmount)
}

func (tester *VestDelegationTester) powerDownRelated(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.NoError(d.ProduceBlocks( int(constants.HardFork3) + 1 ))

	// create a block producer to start the economy system
	name := "testacc"
	makeBp(name,t,d)
	witWrap := d.BlockProducer(name)
	a.True(witWrap.CheckExist())

	// actor0 has 10000.1 vest and power down 5000, actor1 has 0.1 vest
	a.NoError(tester.acc0.SendTrxAndProduceBlock(TransferToVest(tester.acc0.Name, tester.acc0.Name, 10000000000, "")))
	powerDownAmount := 5000000000
	opPowerDown := ConvertVest(tester.acc0.Name, uint64(powerDownAmount))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(opPowerDown))

	// before 5000 vest power down finish, actor0 delegate 10000 vest -> failed
	delegateAmount := uint64(10000000000)
	opDelegation := DelegateVest(tester.acc0.Name, tester.acc1.Name, delegateAmount, constants.MinVestDelegationInBlocks)
	a.Error(tester.acc0.SendTrxAndProduceBlock(opDelegation))

	// actor0 delegate 5000 vest to actor1 and has enough reputation -> success
	//  actor0's vest decrease 5000 and actor1's vest increase 5000 immediately
	delegateAmount = uint64(5000000000)
	oldVestAcc0 := tester.acc0.GetVest()
	oldVestAcc1 := tester.acc1.GetVest()
	opDelegation = DelegateVest(tester.acc0.Name, tester.acc1.Name, delegateAmount, constants.MinVestDelegationInBlocks)
	a.NoError(tester.acc0.SendTrxAndProduceBlock(opDelegation))
	newVestAcc0 := tester.acc0.GetVest()
	newVestAcc1 := tester.acc1.GetVest()
	a.Equal(oldVestAcc0.Value - newVestAcc0.Value, delegateAmount)
	a.Equal(newVestAcc1.Value - oldVestAcc1.Value, delegateAmount)

	// actor0 cancel old power down and delegate rest -> success
	powerDownAmount = 1000000
	opPowerDown = ConvertVest(tester.acc0.Name, uint64(powerDownAmount))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(opPowerDown))
	a.NoError(d.ProduceBlocks( int(constants.HardFork2ConvertWeeks * constants.PowerDownBlockInterval) + 1 ))
	delegateAmount = tester.acc0.GetVest().Value - uint64(powerDownAmount) - constants.DefaultAccountCreateFee
	opDelegation = DelegateVest(tester.acc0.Name, tester.acc1.Name, delegateAmount, constants.MinVestDelegationInBlocks)
	a.NoError(tester.acc0.SendTrxAndProduceBlock(opDelegation))

	// actor1 power down 5000 vest -> failed
	powerDownAmount = 5000000000
	opPowerDown = ConvertVest(tester.acc1.Name, uint64(powerDownAmount))
	a.Error(tester.acc1.SendTrxAndProduceBlock(opPowerDown))

	// after first delegate maturity actor0 undelegate 5000 vest -> success
	// actor1's vest decrease 5000 immediately
	a.NoError(d.ProduceBlocks( int(constants.MinVestDelegationInBlocks) + 1 ))
	oldVestAcc1 = tester.acc1.GetVest()
	opUnDelegation := UnDelegateVest(tester.acc0.Name, 1)
	a.NoError(tester.acc0.SendTrxAndProduceBlock(opUnDelegation))
	newVestAcc1 = tester.acc1.GetVest()
	a.Equal(newVestAcc1.Value + 5000000000, oldVestAcc1.Value)

	// before VestDelegationDeliveryInBlocks actor0 power down 5000 vest -> failed
	powerDownAmount = 5000000000
	opPowerDown = ConvertVest(tester.acc0.Name, uint64(powerDownAmount))
	a.Error(tester.acc0.SendTrxAndProduceBlock(opPowerDown))

	// after VestDelegationDeliveryInBlocks actor0's vest increase 5000
	// actor0 power down -> success
	oldVestAcc0 = tester.acc0.GetVest()
	a.NoError(d.ProduceBlocks( int(constants.VestDelegationDeliveryInBlocks) + 1 ))
	newVestAcc0 = tester.acc0.GetVest()
	a.Equal(oldVestAcc0.Value + 5000000000, newVestAcc0.Value)
	powerDownAmount = 5000000000
	opPowerDown = ConvertVest(tester.acc0.Name, uint64(powerDownAmount))
	a.NoError(tester.acc0.SendTrxAndProduceBlock(opPowerDown))
}

func (tester *VestDelegationTester) multiDelegation(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.NoError(d.ProduceBlocks( int(constants.HardFork3) + 1 ))

	// actor0 has 10000.1 vest, actor1 has 10000.1 vest, actor2 and actor3 has 0.1 vest
	a.NoError(tester.acc0.SendTrxAndProduceBlock(TransferToVest(tester.acc0.Name, tester.acc0.Name, 10000000000, "")))
	a.NoError(tester.acc1.SendTrxAndProduceBlock(TransferToVest(tester.acc1.Name, tester.acc1.Name, 10000000000, "")))

	// actor0 delegate 5000 vest to actor1 -> success
	delegateAmount := uint64(5000000000)
	opDelegation := DelegateVest(tester.acc0.Name, tester.acc1.Name, delegateAmount, constants.MinVestDelegationInBlocks)
	a.NoError(tester.acc0.SendTrxAndProduceBlock(opDelegation))

	// actor0 delegate 5000 vest to actor2 -> success
	delegateAmount = uint64(5000000000)
	opDelegation = DelegateVest(tester.acc0.Name, tester.acc2.Name, delegateAmount, constants.MinVestDelegationInBlocks)
	a.NoError(tester.acc0.SendTrxAndProduceBlock(opDelegation))

	// actor1 delegate 15000 vest to actor2 -> failed
	delegateAmount = uint64(15000000000)
	opDelegation = DelegateVest(tester.acc1.Name, tester.acc2.Name, delegateAmount, constants.MinVestDelegationInBlocks)
	a.Error(tester.acc1.SendTrxAndProduceBlock(opDelegation))

	// actor2 delegate 5000 vest to actor3 -> failed
	delegateAmount = uint64(5000000000)
	opDelegation = DelegateVest(tester.acc2.Name, tester.acc3.Name, delegateAmount, constants.MinVestDelegationInBlocks)
	a.Error(tester.acc2.SendTrxAndProduceBlock(opDelegation))
}

func (tester *VestDelegationTester) shouldRefuseTransaction(refused bool, t *testing.T, acc *DandelionAccount, operations...*prototype.Operation) {
	a := assert.New(t)
	receipt, err := acc.SendTrxEx(operations...)
	if refused {
		a.NotNil(err)
		a.Nil(receipt)
	} else {
		a.Nil(err)
		a.NotNil(receipt)
	}
}
