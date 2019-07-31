package op

import (
	"github.com/coschain/contentos-go/dandelion"
	"testing"
)

func TestOperations(t *testing.T) {
	t.Run("transfer", dandelion.NewDandelionTest(new(TransferTester).Test, 3))
	t.Run("bp", dandelion.NewDandelionTest(new(BpTest).Test, 3))
	t.Run("vote", dandelion.NewDandelionTest(new(VoteTester).TestNormal, 3))
	t.Run("vote", dandelion.NewDandelionTest(new(VoteTester).TestRevote, 3))
	t.Run("vote", dandelion.NewDandelionTest(new(VoteTester).TestFullPower, 3))
	t.Run("follow", dandelion.NewDandelionTest(new(FollowTester).Test, 3))
	t.Run("transfer to vest", dandelion.NewDandelionTest(new(TransferToVestTester).Test, 3))
	t.Run("contract_deploy", dandelion.NewDandelionTest(new(ContractDeployTester).Test, 3))
	t.Run("contract_limits", NewDandelionContractTest(new(ContractLimitsTester).Test, 1, "actor0.limits"))
	t.Run("contract_lib", NewDandelionContractTest(new(ContractTester).Test, 2, "actor0.native_tester", "actor1.native_tester"))
	t.Run("create account", dandelion.NewDandelionTest(new(AccountCreateTester).Test, 3))
	t.Run("convert vest", dandelion.NewDandelionTest(new(ConvertVestTester).Test, 5))
	t.Run("update account", dandelion.NewDandelionTest(new(AccountUpdateTester).Test, 3))
	t.Run("stake", dandelion.NewDandelionTest(new(StakeTester).Test, 3))
	t.Run("unStake", dandelion.NewDandelionTest(new(UnStakeTester).Test, 3))
}

func TestContractGasUsage(t *testing.T) {
	t.Run("contract_gas", NewContractGasTest(123, 1380, 3110))
}
