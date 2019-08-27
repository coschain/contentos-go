package blocklog

import (
	"encoding/json"
	"fmt"
	"github.com/coschain/contentos-go/app/blocklog"
	"github.com/coschain/contentos-go/common/constants"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/tests/op"
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

const sBlockLogTestActors = 10

func TestBlockLog(t *testing.T) {
	t.Run("block_log", NewDandelionTest(new(BlockLogTester).Test, sBlockLogTestActors))
}

type BlockLogTester struct {
	a *assert.Assertions
	d *Dandelion
	actors []*DandelionAccount
}

func (tester *BlockLogTester) Test(t *testing.T, d *Dandelion) {
	tester.d, tester.a = d, assert.New(t)
	for i := 0; i < sBlockLogTestActors; i++ {
		tester.actors = append(tester.actors, d.Account(fmt.Sprintf("actor%d", i)))
	}
	tester.prepare()
	tester.doSomething()
}

func (tester *BlockLogTester) doSomething() {
	tester.a.NoError(tester.d.Account("actor0").SendTrx(Transfer("actor0", "actor1", math.MaxUint64, "")))

	tester.a.NoError(tester.d.Account("actor0").SendTrx(Transfer("actor0", "actor1", 1, "")))
	tester.a.NoError(tester.d.Account("actor2").SendTrx(Transfer("actor2", "actor3", 2, "")))
	tester.a.NoError(tester.d.Account("actor4").SendTrx(Transfer("actor4", "actor5", 3, "")))
	tester.a.NoError(tester.d.Account("actor3").SendTrx(Post(1, "actor3", "title", "content", []string{"test"}, []map[string]int{
		{"actor7": 5000},
		{"actor8": 5000},
	})))
	tester.a.NoError(tester.d.Account("actor1").SendTrx(Vote("actor1", 1)))
	tester.a.NoError(tester.d.Account("actor4").SendTrx(Vote("actor4", 1)))
	tester.a.NoError(tester.d.Account("actor6").SendTrx(Vote("actor6", 1)))
	tester.a.NoError(tester.d.Account("actor0").SendTrx(ContractApply("actor0", "actor0", "token", "create", `["USDollar", "USD", 10000000000, 6]`, 123)))
	tester.a.NoError(tester.d.ProduceBlocks(1))

	tester.a.NoError(tester.d.Account("actor4").SendTrxAndProduceBlock(Reply(2, 1, "actor4",  "content:reply", []map[string]int{
		{"actor7": 5000},
		{"actor8": 5000},
	})))
	tester.a.NoError(tester.d.ProduceBlocks(1))
	tester.a.NoError(tester.d.Account("actor6").SendTrxAndProduceBlock(Vote("actor6", 2)))

	tester.a.NoError(tester.d.Account("actor0").SendTrx(ContractApply("actor0", "actor0", "token", "transfer", `["actor0", "actor1", 8888]`, 0)))
	tester.a.NoError(tester.d.ProduceBlocks(1))

	waits := constants.PostCashOutDelayBlock
	if waits < constants.VoteCashOutDelayBlock {
		waits = constants.VoteCashOutDelayBlock
	}
	tester.a.NoError(tester.d.ProduceBlocks(waits))
}

func (tester *BlockLogTester) prepare() {
	producers := []int{0, 1, 2, 3}
	tester.addBlockProducer(producers...)
	tester.a.NoError(op.Deploy(tester.d, "actor0", "token"))
	tester.a.NoError(tester.d.ProduceBlocks(constants.BlockProdRepetition * len(producers)))
	_ = tester.d.Node().EvBus.Subscribe(constants.NoticeBlockLog, tester.onBlockLog)
}

func (tester *BlockLogTester) addBlockProducer(who...int) {
	a := tester.a
	var ops []*prototype.Operation
	bpInitminer := tester.d.BlockProducer(constants.COSInitMiner)
	chainProp := &prototype.ChainProperties{
		AccountCreationFee:   bpInitminer.GetAccountCreateFee(),
		StaminaFree:          bpInitminer.GetProposedStaminaFree(),
		TpsExpected:          bpInitminer.GetTpsExpected(),
		TopNAcquireFreeToken: bpInitminer.GetTopNAcquireFreeToken(),
		EpochDuration:        bpInitminer.GetEpochDuration(),
		PerTicketPrice:       bpInitminer.GetPerTicketPrice(),
		PerTicketWeight:      bpInitminer.GetPerTicketWeight(),
	}
	ops = append(ops, Stake(constants.COSInitMiner, constants.COSInitMiner, 10000 * constants.COSTokenDecimals))
	for _, which := range who {
		actor := tester.actors[which]
		ops = append(ops, Stake(constants.COSInitMiner, actor.Name, 10000 * constants.COSTokenDecimals))
		ops = append(ops, TransferToVest(constants.COSInitMiner, actor.Name, constants.MinBpRegisterVest, ""))
	}
	r := tester.d.Account(constants.COSInitMiner).TrxReceipt(ops...)
	a.True(r != nil && r.Status == prototype.StatusSuccess)

	for _, which := range who {
		actor := tester.actors[which]
		r := actor.TrxReceipt(
			BpRegister(actor.Name, "http://foo", "blabla", actor.GetPubKey(), chainProp),
			BpVote(actor.Name, actor.Name, false))
		a.True(r != nil && r.Status == prototype.StatusSuccess)
	}
}

func (tester *BlockLogTester) onBlockLog(blockLog *blocklog.BlockLog) {
	tester.a.NotNil(blockLog)
	j, err := json.MarshalIndent(blockLog, "", "    ")
	tester.a.NoError(err)
	fmt.Printf("block log #%d\n%s\n", blockLog.BlockNum, string(j))
}
