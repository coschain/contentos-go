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

var defaultProps *prototype.ChainProperties

func resetProperties(p **prototype.ChainProperties) {
	*p = &prototype.ChainProperties{
		AccountCreationFee: prototype.NewCoin(1),
		MaximumBlockSize:   1024 * 1024,
		StaminaFree:        constants.DefaultStaminaFree,
		TpsExpected:        constants.DefaultTPSExpected,
		EpochDuration:      constants.InitEpochDuration,
		TopNAcquireFreeToken: constants.InitTopN,
		PerTicketPrice:     prototype.NewCoin(1000000),
		PerTicketWeight:    constants.PerTicketWeight,
	}
}

func checkError(r* prototype.TransactionReceiptWithInfo) error {
	if r == nil {
		return errors.New("receipt is nil")
	}
	if r.Status != prototype.StatusSuccess {
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

	ops = append(ops,Stake(constants.COSInitMiner,"actor0",1))
	ops = append(ops,Stake(constants.COSInitMiner,"actor1",1))
	ops = append(ops,Stake(constants.COSInitMiner,"actor2",1))

	if err := checkError(d.Account(constants.COSInitMiner).TrxReceipt(ops...)); err != nil {
		t.Error(err)
		return
	}
	resetProperties(&defaultProps)

	t.Run("regist", d.Test(tester.regist))
	t.Run("registInvalidParam", d.Test(tester.registInvalidParam))
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
	a.NoError(checkError(d.Account(tester.acc0.Name).TrxReceipt(BpDisable(tester.acc0.Name))))
}

func (tester *BpTest) registInvalidParam(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	newBpName := "newwitness"
	// create a new account to be new bp
	priv, _ := prototype.GenerateNewKey()
	pub, _ := priv.PubKey()
	a.NoError(checkError(d.Account(constants.COSInitMiner).TrxReceipt(AccountCreate(constants.COSInitMiner,newBpName,
		pub,1,""))))

	// new bp regist as bp, but he has no 10000 vesting, should failed
	a.Error(checkError(d.Account(newBpName).TrxReceipt(BpRegister(newBpName,"","",pub,defaultProps))))

	// now give new bp 10000 vesting
	a.NoError(checkError(d.Account(constants.COSInitMiner).TrxReceipt(TransferToVest(constants.COSInitMiner,newBpName,constants.MinBpRegisterVest))))

	// set invalid stamina, should failed
	defaultProps.StaminaFree = constants.MaxStaminaFree + 1
	a.Error(checkError(d.Account(newBpName).TrxReceipt(BpRegister(newBpName,"","",pub,defaultProps))))
	defaultProps.StaminaFree = constants.MaxStaminaFree

	// set invalid tps expected, should failed
	defaultProps.TpsExpected = 0
	a.Error(checkError(d.Account(newBpName).TrxReceipt(BpRegister(newBpName,"","",pub,defaultProps))))
	defaultProps.TpsExpected = constants.MinTPSExpected

	// set invalid account create fee, should failed
	defaultProps.AccountCreationFee = prototype.NewCoin(0)
	a.Error(checkError(d.Account(newBpName).TrxReceipt(BpRegister(newBpName,"","",pub,defaultProps))))
	defaultProps.AccountCreationFee = prototype.NewCoin(constants.DefaultAccountCreateFee)

	// set invalid topNFreeToken, should failed
	defaultProps.TopNAcquireFreeToken = constants.MaxTopN + 1
	a.Error(checkError(d.Account(newBpName).TrxReceipt(BpRegister(newBpName,"","",pub,defaultProps))))
	defaultProps.TopNAcquireFreeToken = constants.MaxTopN

	// set invalid ticket price, should failed
	defaultProps.PerTicketPrice = prototype.NewCoin(0)
	a.Error(checkError(d.Account(newBpName).TrxReceipt(BpRegister(newBpName,"","",pub,defaultProps))))
	defaultProps.PerTicketPrice = prototype.NewCoin(constants.MinTicketPrice)

	// acc0 should not appear in bp
	witWrap := d.Witness(newBpName)
	a.False(witWrap.CheckExist())
	resetProperties(&defaultProps)
}

func (tester *BpTest) dupRegist(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	// acc1 regist as bp
	a.NoError(checkError(d.Account(tester.acc1.Name).TrxReceipt(BpRegister(tester.acc1.Name,"www.me.com","nothing",tester.acc1.GetPubKey(),defaultProps))))
	witWrap := d.Witness(tester.acc1.Name)
	a.True(witWrap.CheckExist())

	// acc1 regist again, this time should failed
	a.Error(checkError(d.Account(tester.acc1.Name).TrxReceipt(BpRegister(tester.acc1.Name,"www.you.com","nothing",tester.acc1.GetPubKey(),defaultProps))))
	witWrapCheck := d.Witness(tester.acc1.Name)
	// acc1's bp info should be in old
	a.True(witWrapCheck.GetUrl() == "www.me.com")
}

func (tester *BpTest) bpVote(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.True(tester.acc1.GetBpVoteCount() == 0)

	// acc1 vote for bp acc1
	a.NoError(checkError(d.Account(tester.acc1.Name).TrxReceipt(BpVote(tester.acc1.Name,tester.acc1.Name,false))))

	witWrap := d.Witness(tester.acc1.Name)

	// check bp's vote count and acc1's vote count
	a.True(witWrap.GetVoteVest().Value > 0)
	a.True(tester.acc1.GetBpVoteCount() == 1)
}

func (tester *BpTest) bpUnVote(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.True(tester.acc2.GetBpVoteCount() == 0)

	// acc2 vote for bp
	a.NoError(checkError(d.Account(tester.acc2.Name).TrxReceipt(BpVote(tester.acc2.Name,tester.acc1.Name,false))))

	// check bp's vote count and acc2's vote count
	witWrap := d.Witness(tester.acc1.Name)
	a.True(witWrap.GetVoteVest().Value > 0)
	a.True(tester.acc2.GetBpVoteCount() == 1)

	// acc2 unvote
	a.NoError(checkError(d.Account(tester.acc2.Name).TrxReceipt(BpVote(tester.acc2.Name,tester.acc1.Name,true))))
	// check acc2's vote count
	a.True(tester.acc2.GetBpVoteCount() == 0)
}

func (tester *BpTest) bpVoteMultiTime(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.True(tester.acc2.GetBpVoteCount() == 0)
	witWrap := d.Witness(tester.acc1.Name)
	a.True(witWrap.CheckExist())

	// acc2 vote for bp acc1
	a.NoError(checkError(d.Account(tester.acc2.Name).TrxReceipt(BpVote(tester.acc2.Name,tester.acc1.Name,false))))

	// check acc2's vote count
	witWrap2 := d.Witness(tester.acc1.Name)
	a.True(witWrap2.GetVoteVest().Value > 0)
	a.True(tester.acc2.GetBpVoteCount() == 1)

	// acc2 vote again for bp
	a.Error(checkError(d.Account(tester.acc2.Name).TrxReceipt(BpVote(tester.acc2.Name,tester.acc1.Name,false))))
	// acc2's vote count should stay original
	a.True(tester.acc2.GetBpVoteCount() == 1)
}

// todo 2/3 check?
func (tester *BpTest) bpUpdate(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	// change staminaFree param
	witWrap := d.Witness(tester.acc1.Name)
	a.True(witWrap.GetProposedStaminaFree() == constants.DefaultStaminaFree)
	defaultProps.StaminaFree = 1

	// acc1 update bp property
	a.NoError(checkError(d.Account(tester.acc1.Name).TrxReceipt(BpUpdate(tester.acc1.Name,defaultProps))))

	// check stamina
	witWrap2 := d.Witness(tester.acc1.Name)
	a.True(witWrap2.GetProposedStaminaFree() == 1)
}

func (tester *BpTest) unRegist(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	witWrap := d.Witness(tester.acc1.Name)
	a.True(witWrap.GetActive())

	// acc1 unregist
	a.NoError(checkError(d.Account(tester.acc1.Name).TrxReceipt(BpDisable(tester.acc1.Name))))

	// check status
	a.True(witWrap.CheckExist())
	a.False(witWrap.GetActive())

	// unregist again, should failed
	a.Error(checkError(d.Account(tester.acc1.Name).TrxReceipt(BpDisable(tester.acc1.Name))))
}