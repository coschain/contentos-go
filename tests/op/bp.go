package op

import (
	"github.com/coschain/contentos-go/common/constants"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
	"github.com/kataras/go-errors"
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

func checkError(r* prototype.TransactionReceiptWithInfo) error {
	if r == nil || r.Status != prototype.StatusSuccess {
		return errors.New(r.ErrorInfo)
	}
	return nil
}

func (tester *BpTest) Test(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")
	var ops []*prototype.Operation
	ops = append(ops,TransferToVest(constants.COSInitMiner, "actor0", constants.MinBpRegisterVest))
	ops = append(ops,TransferToVest(constants.COSInitMiner, "actor1", constants.MinBpRegisterVest))
	ops = append(ops,TransferToVest(constants.COSInitMiner, "actor2", constants.MinBpRegisterVest))

	if err := checkError(d.Account(constants.COSInitMiner).TrxReceipt(ops...)); err != nil {
		t.Error(err)
		return
	}

	t.Run("regist", d.Test(tester.regist))
	t.Run("dupRegist", d.Test(tester.dupRegist))
	t.Run("bpVote", d.Test(tester.bpVote))
	t.Run("bpUnVote", d.Test(tester.bpUnVote))
	t.Run("bpVoteMultiTime", d.Test(tester.bpVoteMultiTime))
	t.Run("bpUpdate", d.Test(tester.bpUpdate))
	t.Run("unRegist", d.Test(tester.unRegist))
}

func (tester *BpTest) regist(t *testing.T, d *Dandelion) {
	a := assert.New(t)
    // acc0 regist as bp
	a.NoError(checkError(d.Account(tester.acc0.Name).TrxReceipt(BpRegister(tester.acc0.Name,"www.me.com","nothing",tester.acc0.GetPubKey(),defaultProps))))

	// acc0 should appear in bp
	witWrap := d.Witness(tester.acc0.Name)
	a.True(witWrap.CheckExist())

	// unregist acc0
	a.NoError(checkError(d.Account(tester.acc0.Name).TrxReceipt(BpUnregister(tester.acc0.Name))))
}

func (tester *BpTest) dupRegist(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	// acc0 regist as bp
	a.NoError(checkError(d.Account(tester.acc0.Name).TrxReceipt(BpRegister(tester.acc0.Name,"www.me.com","nothing",tester.acc0.GetPubKey(),defaultProps))))
	witWrap := d.Witness(tester.acc0.Name)
	a.True(witWrap.CheckExist())

	// acc0 regist again, this time should failed
	a.Error(checkError(d.Account(tester.acc0.Name).TrxReceipt(BpRegister(tester.acc0.Name,"www.you.com","nothing",tester.acc0.GetPubKey(),defaultProps))))
	witWrapCheck := d.Witness(tester.acc0.Name)
	// acc0's bp info should be in old
	a.True(witWrapCheck.GetUrl() == "www.me.com")
}

func (tester *BpTest) bpVote(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.True(tester.acc1.GetBpVoteCount() == 0)

	// acc1 vote for bp
	a.NoError(checkError(d.Account(tester.acc1.Name).TrxReceipt(BpVote(tester.acc1.Name,tester.acc0.Name,false))))

	witWrap := d.Witness(tester.acc0.Name)

	// check bp's vote count and acc1's vote count
	a.True(witWrap.GetVoteVest().Value > 0)
	a.True(tester.acc1.GetBpVoteCount() == 1)
}

func (tester *BpTest) bpUnVote(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.True(tester.acc2.GetBpVoteCount() == 0)

	// acc2 vote for bp
	a.NoError(checkError(d.Account(tester.acc2.Name).TrxReceipt(BpVote(tester.acc2.Name,tester.acc0.Name,false))))

	// check bp's vote count and acc2's vote count
	witWrap := d.Witness(tester.acc0.Name)
	a.True(witWrap.GetVoteVest().Value > 0)
	a.True(tester.acc2.GetBpVoteCount() == 1)

	// acc2 unvote
	a.NoError(checkError(d.Account(tester.acc2.Name).TrxReceipt(BpVote(tester.acc2.Name,tester.acc0.Name,true))))
	// check acc2's vote count
	a.True(tester.acc2.GetBpVoteCount() == 0)
}

func (tester *BpTest) bpVoteMultiTime(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.True(tester.acc2.GetBpVoteCount() == 0)

	// acc2 vote for bp
	a.NoError(checkError(d.Account(tester.acc2.Name).TrxReceipt(BpVote(tester.acc2.Name,tester.acc0.Name,false))))

	// check acc2's vote count
	witWrap := d.Witness(tester.acc0.Name)
	a.True(witWrap.GetVoteVest().Value > 0)
	a.True(tester.acc2.GetBpVoteCount() == 1)

	// acc2 vote again for bp
	a.Error(checkError(d.Account(tester.acc2.Name).TrxReceipt(BpVote(tester.acc2.Name,tester.acc0.Name,false))))
	// acc2's vote count should stay original
	a.True(tester.acc2.GetBpVoteCount() == 1)
}

// todo 2/3 check?
func (tester *BpTest) bpUpdate(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	// change staminaFree param
	witWrap := d.Witness(tester.acc0.Name)
	a.True(witWrap.GetProposedStaminaFree() == constants.DefaultStaminaFree)
	defaultProps.StaminaFree = 1

	// acc0 update bp property
	a.NoError(checkError(d.Account(tester.acc0.Name).TrxReceipt(BpUpdate(tester.acc0.Name,defaultProps))))

	// check stamina
	witWrap2 := d.Witness(tester.acc0.Name)
	a.True(witWrap2.GetProposedStaminaFree() == 1)
}

func (tester *BpTest) unRegist(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	witWrap := d.Witness(tester.acc0.Name)
	a.True(witWrap.GetActive())

	// acc0 unregist
	a.NoError(checkError(d.Account(tester.acc0.Name).TrxReceipt(BpUnregister(tester.acc0.Name))))

	// check status
	a.True(witWrap.CheckExist())
	a.False(witWrap.GetActive())

	// unregist again, should failed
	a.Error(checkError(d.Account(tester.acc0.Name).TrxReceipt(BpUnregister(tester.acc0.Name))))
}