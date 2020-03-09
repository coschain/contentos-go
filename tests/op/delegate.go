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
