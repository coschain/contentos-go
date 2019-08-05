package op

import (
	"fmt"
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

//var defaultProps *prototype.ChainProperties

// << start of helper function

func makeBPChainProperty() *prototype.ChainProperties {
	return &prototype.ChainProperties{
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

func newAccount(name string,t *testing.T,d *Dandelion) *prototype.PrivateKeyType{
	a := assert.New(t)
	priv, _ := prototype.GenerateNewKey()
	pub, _ := priv.PubKey()
	a.NoError(checkError(d.Account(constants.COSInitMiner).TrxReceipt(AccountCreate(constants.COSInitMiner,name,
		pub,1,""))))
	d.PutAccount(name,priv)
	return priv
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

func makeBp(name string,t *testing.T,d *Dandelion) {
	a := assert.New(t)
	pri := newAccount(name,t,d)
	pub,_ := pri.PubKey()

	// give new bp 10000 vesting
	a.NoError(checkError(d.Account(constants.COSInitMiner).TrxReceipt(TransferToVest(constants.COSInitMiner,name,constants.MinBpRegisterVest))))

	a.NoError(checkError(d.Account(name).TrxReceipt(BpRegister(name,"","",pub,makeBPChainProperty()))))
}

func stakeToInitMiner(t *testing.T, d *Dandelion) {
	var ops []*prototype.Operation
	ops = append(ops,Stake(constants.COSInitMiner,constants.COSInitMiner,1))
	if err := checkError(d.Account(constants.COSInitMiner).TrxReceipt(ops...)); err != nil {
		t.Error(err)
		return
	}
}

// >> end of helper function

func (tester *BpTest) TestNormal(t *testing.T, d *Dandelion) {

	t.Run("regist", d.Test(tester.regist))
	t.Run("registInvalidParam", d.Test(tester.registInvalidParam))
	t.Run("bpVote", d.Test(tester.bpVote))
	t.Run("bpVoteNoExist", d.Test(tester.bpVoteNoExist))
	t.Run("bpVoteNoBp",d.Test(tester.bpVoteNoBp))
	t.Run("bpVoteDisableBp",d.Test(tester.bpVoteDisableBp))
	t.Run("bpUnVote", d.Test(tester.bpUnVote))
	t.Run("bpUpdate", d.Test(tester.bpUpdate))
	t.Run("bpEnableDisable", d.Test(tester.bpEnableDisable))
}

func (tester *BpTest) TestDuplicate(t *testing.T, d *Dandelion) {

	t.Run("enableDup", d.Test(tester.enableDup))
	t.Run("registDup", d.Test(tester.registDup))
	t.Run("bpVoteDup", d.Test(tester.bpVoteDup))
	t.Run("bpUnVoteDup", d.Test(tester.bpUnVoteDup))
	t.Run("disableDup", d.Test(tester.disableDup))
}

func (tester *BpTest) TestGlobalProperty(t *testing.T, d *Dandelion) {
	t.Run("bpUpdateCheckDgp", d.Test(tester.bpUpdateCheckDgp))
}

func (tester *BpTest) TestSwitch(t *testing.T, d *Dandelion) {
	stakeToInitMiner(t,d)
	t.Run("bpSwitch", d.Test(tester.bpSwitch))
}

func (tester *BpTest) enableDup(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	bpName := "enableDup"
	makeBp(bpName,t,d)

	// enable duplicate, should failed
	a.Error(checkError(d.Account(bpName).TrxReceipt(BpEnable(bpName))))
}

func (tester *BpTest) registDup(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	bpName := "registDupBp"
	pri := newAccount(bpName,t,d)
	pub,_ := pri.PubKey()


	props := makeBPChainProperty()
	// regist as bp
	a.NoError(checkError(d.Account(constants.COSInitMiner).TrxReceipt(TransferToVest(constants.COSInitMiner,bpName,constants.MinBpRegisterVest))))
	a.NoError(checkError(d.Account(bpName).TrxReceipt(BpRegister(bpName,"www.me.com","nothing",pub,props))))
	witWrap := d.BlockProducer(bpName)
	a.True(witWrap.CheckExist())

	// regist again, this time should failed
	a.Error(checkError(d.Account(bpName).TrxReceipt(BpRegister(bpName,"www.you.com","nothing",pub,props))))
	witWrapCheck := d.BlockProducer(bpName)
	// bp info should be in old
	a.True(witWrapCheck.GetUrl() == "www.me.com")
}

func (tester *BpTest) bpVoteDup(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	bpName := "newBpVoteDup"
	makeBp(bpName,t,d)

	voter := "bpVoteDupVoter"
	newAccount(voter,t,d)

	// vote for bp
	a.NoError(checkError(d.Account(voter).TrxReceipt(BpVote(voter,bpName,false))))

	// check voter's vote count
	witWrap2 := d.BlockProducer(bpName)
	a.True(witWrap2.GetBpVest().VoteVest.Value > 0)
	a.True(d.Account(voter).GetBpVoteCount() == 1)

	// voter vote again for bp
	a.Error(checkError(d.Account(voter).TrxReceipt(BpVote(voter,bpName,false))))
	// voter's vote count should stay original
	a.True(d.Account(voter).GetBpVoteCount() == 1)
}

func (tester *BpTest) regist(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	newBpName := "registBp"
	makeBp(newBpName,t,d)

	// should appear in bp
	witWrap := d.BlockProducer(newBpName)
	a.True(witWrap.CheckExist())
}

func (tester *BpTest) registInvalidParam(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	// create a new account to be new bp
	newBpName := "newwitness"
	pri := newAccount(newBpName,t,d)
	pub,_ := pri.PubKey()

	props := makeBPChainProperty()
	// new bp regist as bp, but he has no 10000 vesting, should failed
	a.Error(checkError(d.Account(newBpName).TrxReceipt(BpRegister(newBpName,"","",pub,props))))

	// now give new bp 10000 vesting
	a.NoError(checkError(d.Account(constants.COSInitMiner).TrxReceipt(TransferToVest(constants.COSInitMiner,newBpName,constants.MinBpRegisterVest))))

	// set invalid stamina, should failed
	props.StaminaFree = constants.MaxStaminaFree + 1
	a.Error(checkError(d.Account(newBpName).TrxReceipt(BpRegister(newBpName,"","",pub,props))))
	props.StaminaFree = constants.MaxStaminaFree

	// set invalid tps expected, should failed
	props.TpsExpected = 0
	a.Error(checkError(d.Account(newBpName).TrxReceipt(BpRegister(newBpName,"","",pub,props))))
	props.TpsExpected = constants.MinTPSExpected

	// set invalid account create fee, should failed
	props.AccountCreationFee = prototype.NewCoin(0)
	a.Error(checkError(d.Account(newBpName).TrxReceipt(BpRegister(newBpName,"","",pub,props))))
	props.AccountCreationFee = prototype.NewCoin(constants.DefaultAccountCreateFee)

	// set invalid topNFreeToken, should failed
	props.TopNAcquireFreeToken = constants.MaxTopN + 1
	a.Error(checkError(d.Account(newBpName).TrxReceipt(BpRegister(newBpName,"","",pub,props))))
	props.TopNAcquireFreeToken = constants.MaxTopN

	// set invalid ticket price, should failed
	props.PerTicketPrice = prototype.NewCoin(0)
	a.Error(checkError(d.Account(newBpName).TrxReceipt(BpRegister(newBpName,"","",pub,props))))
	props.PerTicketPrice = prototype.NewCoin(constants.MinTicketPrice)

	// new account should not appear in bp
	witWrap := d.BlockProducer(newBpName)
	a.False(witWrap.CheckExist())
}

func (tester *BpTest) bpVote(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	bpName := "bpVote"
	makeBp(bpName,t,d)

	voteName := "bpVoteVoter"
	newAccount("bpVoteVoter",t,d)
	// voteName vote for bp
	a.NoError(checkError(d.Account(voteName).TrxReceipt(BpVote(voteName,bpName,false))))

	witWrap := d.BlockProducer(bpName)
	// check bp's vote count and bpName's vote count
	a.True(witWrap.GetBpVest().VoteVest.Value > 0)
	a.True(d.Account(voteName).GetBpVoteCount() == 1)
}

func (tester *BpTest) bpVoteNoExist(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	name := "bpVoteNoExist"
	newAccount(name,t,d)

	noExistName := "actor10"

	// vote for account no exist, should failed
	a.Error(checkError(d.Account(name).TrxReceipt(BpVote(name,noExistName,false))))
}

func (tester *BpTest) bpVoteNoBp(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	name := "bpVoteNoBp"
	newAccount(name,t,d)

	bpName := "bpVoteNoBpBp"
	newAccount(bpName,t,d)

	// vote for newaccount,but newaccount is not bp, should failed
	a.Error(checkError(d.Account(name).TrxReceipt(BpVote(name,bpName,false))))
}

func (tester *BpTest) bpVoteDisableBp(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	bpName := "bpVoteDisableBp"
	makeBp(bpName,t,d)

	// bpName disable
	a.NoError(checkError(d.Account(bpName).TrxReceipt(BpDisable(bpName))))
	witWrap := d.BlockProducer(bpName)
	a.False(witWrap.GetBpVest().Active)

	name := "bpVoteDisable"
	newAccount(name,t,d)
	// vote for disable bp, should failed
	a.Error(checkError(d.Account(name).TrxReceipt(BpVote(name,bpName,false))))
}

func (tester *BpTest) bpUnVote(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	bpName := "bpUnVote"
	makeBp(bpName,t,d)

	voter := "bpUnVoteVoter"
	newAccount(voter,t,d)

	// vote for bp
	a.NoError(checkError(d.Account(voter).TrxReceipt(BpVote(voter,bpName,false))))

	// check bp's vote count and voter's vote count
	witWrap := d.BlockProducer(bpName)
	a.True(witWrap.GetBpVest().VoteVest.Value  > 0)
	a.True(d.Account(voter).GetBpVoteCount() == 1)

	// unvote
	a.NoError(checkError(d.Account(voter).TrxReceipt(BpVote(voter,bpName,true))))
	// check voter's vote count
	a.True(d.Account(voter).GetBpVoteCount() == 0)
}

func (tester *BpTest) bpUpdate(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	bpName := "bpUpdate"
	makeBp(bpName,t,d)

	// change staminaFree param
	witWrap := d.BlockProducer(bpName)
	a.True(witWrap.GetProposedStaminaFree() == constants.DefaultStaminaFree)

	props := makeBPChainProperty()
	props.StaminaFree = constants.MaxStaminaFree

	// bpName update bp property
	a.NoError(checkError(d.Account(bpName).TrxReceipt(BpUpdate(bpName,props))))

	// check stamina
	witWrap2 := d.BlockProducer(bpName)
	a.True(witWrap2.GetProposedStaminaFree() == constants.MaxStaminaFree)
}

func (tester *BpTest) bpEnableDisable(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	bpName := "EnableDisable"
	makeBp(bpName,t,d)

	accountName := "EnableDisableA"
	newAccount(accountName,t,d)

	// account not a bp, enable should failed
	a.Error(checkError(d.Account(accountName).TrxReceipt(BpEnable(accountName))))

	// account not a bp, disable should failed
	a.Error(checkError(d.Account(accountName).TrxReceipt(BpDisable(accountName))))

	// bpName is a bp, disable should ok
	a.NoError(checkError(d.Account(bpName).TrxReceipt(BpDisable(bpName))))

	// bpName is a bp, enable should ok
	a.NoError(checkError(d.Account(bpName).TrxReceipt(BpEnable(bpName))))
}

func (tester *BpTest) disableDup(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	bpName := "bpDisableDup"
	makeBp(bpName,t,d)
	// unregist
	a.NoError(checkError(d.Account(bpName).TrxReceipt(BpDisable(bpName))))

	// unregist again, should failed
	a.Error(checkError(d.Account(bpName).TrxReceipt(BpDisable(bpName))))
}

func (tester *BpTest) bpUpdateCheckDgp(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	name := "blockproducer"
	tps := uint64(constants.MinTPSExpected)
	tpsStart := tps
	// create 21 bp
	for i:=0;i<constants.MaxBlockProducerCount;i++ {
		tmpName := name + fmt.Sprintf("%d",i)
		pri := newAccount(tmpName,t,d)
		pub,_ := pri.PubKey()

		// give new bp 10000 vesting
		a.NoError(checkError(d.Account(constants.COSInitMiner).TrxReceipt(TransferToVest(constants.COSInitMiner,tmpName,constants.MinBpRegisterVest))))

		props := makeBPChainProperty()
		props.TpsExpected = tps
		a.NoError(checkError(d.Account(tmpName).TrxReceipt(BpRegister(tmpName,"","",pub,props))))
		tps++
	}

	// produce some blocks wait shuffle happen to let bp's param take effective
	d.ProduceBlocks(constants.MaxBlockProducerCount)

	// should be median number
	a.True(d.GlobalProps().TpsExpected == tpsStart + constants.MaxBlockProducerCount/2)
}

func (tester *BpTest) bpUnVoteDup(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	bpName := "bpUnVoteMulti"
	pri := newAccount(bpName,t,d)
	pub,_ := pri.PubKey()
	// give new bp 10000 vesting
	a.NoError(checkError(d.Account(constants.COSInitMiner).TrxReceipt(TransferToVest(constants.COSInitMiner,bpName,constants.MinBpRegisterVest))))
	// new account regist bp
	a.NoError(checkError(d.Account(bpName).TrxReceipt(BpRegister(bpName,"","",pub,makeBPChainProperty()))))

	voter := "bpUnVoteMultiv"
	newAccount(voter,t,d)
	// voter vote for new account bp
	a.NoError(checkError(d.Account(voter).TrxReceipt(BpVote(voter,bpName,false))))

	// check voter's vote count
	witWrap := d.BlockProducer(bpName)
	a.True(witWrap.GetBpVest().VoteVest.Value  > 0)
	a.True(d.Account(voter).GetBpVoteCount() == 1)

	// voter vote cancel vote for bp
	a.NoError(checkError(d.Account(voter).TrxReceipt(BpVote(voter,bpName,true))))
	// bp's vote vest should be o
	a.True(witWrap.GetBpVest().VoteVest.Value  == 0)
	// voter's vote count should be 0
	a.True(d.Account(voter).GetBpVoteCount() == 0)

	// voter vote cancel vote again, should failed
	a.Error(checkError(d.Account(voter).TrxReceipt(BpVote(voter,bpName,true))))
}

func (tester *BpTest) bpSwitch(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	topN := uint32(21)
	// get old bp list
	bpArray,_ := d.GetBlockProducerTopN(topN)
	oldBpMap := map[string]bool{}
	for _,bp := range bpArray{
		oldBpMap[bp] = true
	}

	// we make 21 new bp, and intend to replace old bp
	newBpMap := map[string]bool{}
	newNames := []string{}
	name := "newproducer"
	for i:=0;i<constants.MaxBlockProducerCount;i++ {
		tmpName := name + fmt.Sprintf("%d",i)
		makeBp(tmpName,t,d)
		newNames = append(newNames,tmpName)
		newBpMap[tmpName] = true
	}

	// create 42 new accounts to vote for new bp
	voterName := "votergroup1"
	for i:=0;i<constants.MaxBlockProducerCount*2;i++ {
		if i == constants.MaxBlockProducerCount {
			voterName = "votergroup2"
		}
		tmpName := voterName + fmt.Sprintf("%d",i)
		newAccount(tmpName,t,d)
		a.NoError(checkError(d.Account(tmpName).TrxReceipt(BpVote(tmpName,newNames[i%21],false))))
	}

	bpArray,_ = d.GetBlockProducerTopN(topN)
	a.True(uint32(len(bpArray)) == topN)

	for _,bp := range bpArray {
		// new bp list should not contain old bp
		a.False(oldBpMap[bp])
	}

	// we make another 21 new bp, these bps have no votes so they should not appear in producer list
	newBpNoVoteMap := map[string]bool{}
	nameBpNoVote := "withoutvote"
	for i:=0;i<constants.MaxBlockProducerCount;i++ {
		tmpName := nameBpNoVote + fmt.Sprintf("%d",i)
		makeBp(tmpName,t,d)
		newBpNoVoteMap[tmpName] = true
	}

	// get 21 producer again
	bpArray,_ = d.GetBlockProducerTopN(topN)
	a.True(uint32(len(bpArray)) == topN)
	for _,bp := range bpArray {
		// these no vote bp should not appear in producers
		a.False(newBpNoVoteMap[bp])
	}

	// make a bp that has many votes, should appear in first place of bp list
	firstBpName := "ihavemanyvote"
	makeBp(firstBpName,t,d)
	voterName = "votergroup3"
	for i:=0;i<constants.MaxBlockProducerCount;i++ {
		tmpName := voterName + fmt.Sprintf("%d",i)
		newAccount(tmpName,t,d)
		a.NoError(checkError(d.Account(tmpName).TrxReceipt(BpVote(tmpName,firstBpName,false))))
	}
	bpArray,_ = d.GetBlockProducerTopN(topN)
	a.True(uint32(len(bpArray)) == topN)
	// first place should be firstBpName
	a.True(bpArray[0] == firstBpName)

	// let firstBpName change to disable, then firstBpName should not appear in bp list
	a.NoError(checkError(d.Account(firstBpName).TrxReceipt(BpDisable(firstBpName))))
	bpArray,_ = d.GetBlockProducerTopN(topN)
	a.True(uint32(len(bpArray)) == topN)
	for _,bp := range bpArray {
		a.True(bp != firstBpName)
	}
}