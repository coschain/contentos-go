package op

import (
	"github.com/coschain/contentos-go/common/constants"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
	"github.com/stretchr/testify/assert"
	"testing"
)

type BpTest struct {
	acc0,acc1,acc2 *DandelionAccount
}

var defaultProps = &prototype.ChainProperties{
	AccountCreationFee: prototype.NewCoin(1),
	MaximumBlockSize:   1024 * 1024,
	StaminaFree:        constants.DefaultStaminaFree,
	TpsExpected:        constants.DefaultTPSExpected,
	EpochDuration:      constants.InitEpochDuration,
	TopNAcquireFreeToken: constants.InitTopN,
	PerTicketPrice:     prototype.NewCoin(1000000),
	PerTicketWeight:    constants.PerTicketWeight,
}

func (tester *BpTest) Test(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	t.Run("regist", d.Test(tester.regist))
	t.Run("dupRegist", d.Test(tester.dupRegist))
	t.Run("unRegist", d.Test(tester.unRegist))
	t.Run("bpVote", d.Test(tester.bpVote))
	t.Run("bpVoteMultiTime", d.Test(tester.bpVoteMultiTime))
	t.Run("bpUpdate", d.Test(tester.bpUpdate))
}

func (tester *BpTest) regist(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	a.NoError(tester.acc0.SendTrx(BpRegister(tester.acc0.Name,"www.me.com","nothing",tester.acc0.GetOwner(),defaultProps)))
	a.NoError(d.ProduceBlocks(1))

	witWrap := d.Witness(tester.acc0.Name)
	a.True(witWrap.CheckExist())
}

func (tester *BpTest) dupRegist(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	a.NoError(tester.acc0.SendTrx(BpRegister(tester.acc0.Name,"www.me.com","nothing",tester.acc0.GetOwner(),defaultProps)))
	a.NoError(d.ProduceBlocks(1))
	witWrap := d.Witness(tester.acc0.Name)
	a.True(witWrap.CheckExist())

	a.NoError(tester.acc0.SendTrx(BpRegister(tester.acc0.Name,"www.you.com","nothing",tester.acc0.GetOwner(),defaultProps)))
	a.NoError(d.ProduceBlocks(1))
	witWrapCheck := d.Witness(tester.acc0.Name)
	// should be old witness
	a.True(witWrapCheck.GetUrl() == "www.me.com")
}

func (tester *BpTest) bpVote(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.True(tester.acc1.GetBpVoteCount() == 0)
	a.NoError(tester.acc1.SendTrx(BpVote(tester.acc1.Name,tester.acc0.Name)))
	a.NoError(d.ProduceBlocks(1))
	witWrap := d.Witness(tester.acc0.Name)
	a.True(witWrap.GetVoteCount().Value > 0)
	a.True(tester.acc1.GetBpVoteCount() == 1)
}

func (tester *BpTest) bpVoteMultiTime(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.True(tester.acc2.GetBpVoteCount() == 0)
	a.NoError(tester.acc2.SendTrx(BpVote(tester.acc2.Name,tester.acc0.Name)))
	a.NoError(d.ProduceBlocks(1))
	witWrap := d.Witness(tester.acc0.Name)
	a.True(witWrap.GetVoteCount().Value > 0)
	a.True(tester.acc2.GetBpVoteCount() == 1)

	a.NoError(tester.acc2.SendTrx(BpVote(tester.acc2.Name,tester.acc0.Name)))
	a.NoError(d.ProduceBlocks(1))
	a.True(tester.acc2.GetBpVoteCount() == 1)
}

func (tester *BpTest) bpUpdate(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	witWrap := d.Witness(tester.acc0.Name)
	a.True(witWrap.GetProposedStaminaFree() == constants.DefaultStaminaFree)
	defaultProps.StaminaFree = 1
	a.NoError(tester.acc0.SendTrx(BpUpdate(tester.acc0.Name,defaultProps)))
	a.NoError(d.ProduceBlocks(1))

	witWrap2 := d.Witness(tester.acc0.Name)
	a.True(witWrap2.GetProposedStaminaFree() == 1)
}

func (tester *BpTest) unRegist(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	witWrap := d.Witness(tester.acc0.Name)
	a.True(witWrap.GetActive())
	a.NoError(tester.acc0.SendTrx(BpUnregister(tester.acc0.Name)))
	a.NoError(d.ProduceBlocks(1))
	a.True(witWrap.CheckExist())
	a.False(witWrap.GetActive())
}