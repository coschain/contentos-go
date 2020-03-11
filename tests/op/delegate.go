package op

import (
	"github.com/coschain/contentos-go/common/constants"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
	"github.com/stretchr/testify/assert"
	"testing"
)

type VestDelegationTester struct {
	acc0, acc1, acc2, acc3, acc4  *DandelionAccount
}

func (tester *VestDelegationTester) Test(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")
	tester.acc3 = d.Account("actor3")
	tester.acc4 = d.Account("actor4")

// ===== test cases list
// case 1: hardFork3Switch
//         ===== before hardFork3 block number, although all nodes have been upgraded, delegate or undelegate vest transaction still illegal
//
// case 2: base function test
//         ===== actor0 has 10000.1 vest, actor1 has 0.1 vest, actor2 has 0.1 vest
//         1). actor0 delegate 0 vest -> failed
//         2). actor0 delegate 20000 vest -> failed
//         3). actor0 delegate 10000.1 vest -> failed
//         4). actor0 delegate to himself -> failed
//         5). actor0 delegate 10000 vest to actor1 but his reputation is 0  -> failed
//         6). actor0 delegate 10000 vest to actor1 and has enough reputation -> success
//         ===== actor0's vest decrease 10000 and actor1's vest increase 10000 immediately
//         7). actor1 delegate 10000 vest to actor2 -> failed
//         8). before maturity actor0 undelegate -> failed
//         9). after maturity actor0 undelegate 10000 vest -> success
//         ===== actor1's vest decrease 10000 immediately and actor0's vest not change,
//         ===== before 7 days actor0 delegate 10000 vest -> failed
//         ===== after 7 days actor0's vest increase 10000
//         10). after maturity actor0 undelegate and use the budget id which has already been undelegated -> failed
//
// case 3: bp related
//         ===== actor0 has 10000.1 vest， actor1 has 0.1 vest
//         ===== actor3 and actor4 are producers, actor0 vote actor3, actor1 vote actor4
//         1). actor0 delegate 10000 vest to actor1 and has enough reputation -> success
//         ===== vote count of actor3 decrease 10000 and vote count of actor4 increase 10000 immediately
//         2). after maturity actor0 undelegate 10000 vest -> success
//         ===== vote count of actor4 decrease 10000 immediately and vote count of actor3 not change
//         ===== before 7 days actor1 vote actor3, vote count of actor3 increase only 0.1 vest
//         ===== after 7 days vote count of actor4 increase 10000
//
// case 4: power down related
//         ===== actor0 has 10000.1 vest and power down 5000, actor1 has 0.1 vest
//         ===== before 5000 vest power down finish
//         1). actor0 delegate 10000 vest -> failed
//         2). actor0 delegate 5000 vest to actor1 and has enough reputation -> success
//         ===== actor0's vest decrease 5000 and actor1's vest increase 5000 immediately
//         3). actor0 cancel old power down and delegate rest -> success
//         4). actor1 power down 5000 vest -> failed
//         5). after first delegate maturity actor0 undelegate 5000 vest -> success
//         ===== actor1's vest decrease 5000 immediately
//         6). before 7 days actor0 power down 5000 vest -> failed
//         ===== after 7 days actor0's vest increase 5000
//         7). actor0 power down -> success
//
// case 5: multi delegate
//         ===== actor0 has 10000.1 vest, actor1 has 10000.1 vest, actor2 and actor3 has 0.1 vest
//         1). actor0 delegate 5000 vest to actor1 -> success
//         ===== actor0's vest decrease 5000 and actor1's vest increase 5000 immediately
//         2). actor0 delegate 5000 vest to actor2 -> success
//         ===== actor0's vest decrease 5000 and actor2's vest increase 5000 immediately
//         3). actor1 delegate 15000 vest to actor2 -> failed
//         4). actor2 delegate 5000 vest to actor3 -> failed


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
