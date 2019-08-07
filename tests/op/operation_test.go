package op

import (
	"fmt"
	"github.com/coschain/contentos-go/dandelion"
	"testing"
)

func TestOperations(t *testing.T) {
	t.Run("transfer", dandelion.NewDandelionTest(new(TransferTester).Test, 3))
	t.Run("bp", dandelion.NewDandelionTest(new(BpTest).TestNormal, 0))
	t.Run("bp", dandelion.NewDandelionTest(new(BpTest).TestDuplicate, 0))
	t.Run("bp", dandelion.NewDandelionTest(new(BpTest).TestGlobalProperty, 0))
	t.Run("bp", dandelion.NewDandelionTest(new(BpTest).TestSwitch, 0))
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
	t.Run("ticket", dandelion.NewDandelionTest(new(TicketTester).Test, 3))
	t.Run("post", dandelion.NewDandelionTest(new(PostTest).Test, 1))
	t.Run("reply", dandelion.NewDandelionTest(new(ReplyTest).Test, 3))
}

func TestContractGasUsage(t *testing.T) {
	expected := map[uint32]uint64{
		123: 3101,
		127770551: 3038,
		1737914878: 2555,
		986162247: 3771,
		2056263464: 2934,
		586611393: 4026,
		1866632795: 3068,
		996229099: 3064,
		1919593611: 2860,
		2046711939: 3627,
	}
	for seed, cpu := range expected {
		t.Run(fmt.Sprintf("contract_gas_%d", seed), NewContractGasTest(seed, cpu))
	}
}
