package op

import (
	"fmt"
	"github.com/coschain/contentos-go/common/constants"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
	"github.com/stretchr/testify/assert"
	"testing"
)

type ReputationTester struct {
	acc0, acc1, acc2, acc3, acc4, acc5, acc6, acc7  *DandelionAccount
	bpNum, threshold int
	bpList []*DandelionAccount
}

const repCrtName = "reputation"


func (tester* ReputationTester)Test(t *testing.T, d *Dandelion)  {
	a := assert.New(t)
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")
	tester.acc3 = d.Account("actor3")
	tester.acc4 = d.Account("actor4")
	tester.acc5 = d.Account("actor5")
	tester.acc6 = d.Account("actor6")
	tester.acc7 = d.Account("actor7")

	a.NoError(DeploySystemContract(constants.COSSysAccount, repCrtName, d))
	a.True(d.Contract(constants.COSSysAccount, repCrtName).CheckExist())
	bpList := []*DandelionAccount{tester.acc0, tester.acc1, tester.acc2, tester.acc3}
	if a.NoError(RegisterBp(bpList, d)) {
		tester.bpNum = len(bpList)
		tester.threshold = tester.bpNum/3*2 + 1
		tester.bpList = append(tester.bpList, bpList...)
	}

	t.Run("non bp proposal admin", d.Test(tester.nonBpProposalAdmin))
	t.Run("proposal non exist account", d.Test(tester.proposalWrongBp))
	t.Run("insufficient Bp vote", d.Test(tester.insufficientBpVote))
	t.Run("non bp vote", d.Test(tester.nonBpVote))
	t.Run("reach threshold but expire", d.Test(tester.reachThresholdWhenExpire))
	t.Run("repeat vote", d.Test(tester.repeatVote))
	t.Run("multiple proposals", d.Test(tester.multipleProposals))
	t.Run("set admin success", d.Test(tester.setAdminSuccess))
	t.Run("caller is not admin", d.Test(tester.nonAdminCall))
	t.Run("call wrong contract", d.Test(tester.callWrongContract))
	t.Run("modify non exist account", d.Test(tester.mdNotExiAcct))
	t.Run("modified reputation overFlow", d.Test(tester.repOverFlow))
	t.Run("successfully modify reputation", d.Test(tester.successMdRep))
	t.Run("register bp when reputation is min", d.Test(tester.regBpWithMinRep))
	t.Run("account with min reputation do not cashout", d.Test(tester.minReputationCashout))
	t.Run("vote when reputation is min", d.Test(tester.voteWithMinRep))
	t.Run("use ticket vote when reputation is min", d.Test(tester.voteByTicketWithMinRep))
	t.Run("modify memo to empty", d.Test(tester.mdRepMemoToEmptyStr))
	t.Run("update admin", d.Test(tester.updateAdmin))

}



func (tester *ReputationTester) nonBpProposalAdmin(t *testing.T, d *Dandelion)  {
	NonBpProposalAdmin(t, d, tester.acc4.Name, tester.acc0.Name, repCrtName)
}


func (tester *ReputationTester) proposalWrongBp(t *testing.T, d *Dandelion) {
	ProposalWrongBp(t, d, tester.acc0.Name, repCrtName)
}

//
//all voter is bp but less than 2/3
//
func (tester *ReputationTester) insufficientBpVote(t *testing.T, d *Dandelion) {
	InsufficientBpVote(t, d, repCrtName, tester.acc4.Name, tester.bpList)
}

//
// total vote count more than 2/3,but some voters is't bp
//
func (tester *ReputationTester) nonBpVote(t *testing.T, d *Dandelion) {
	NonBpVote(t, d, repCrtName, tester.acc4.Name, tester.bpList, []*DandelionAccount{tester.acc4})
}

//
// the bp vote count greater than 2/3 but the proposal has expired
//
func (tester *ReputationTester) reachThresholdWhenExpire(t *testing.T, d *Dandelion) {
	ReachThresholdWhenExpire(t, d, repCrtName, tester.acc4.Name,tester.bpList)
}

func (tester *ReputationTester) repeatVote(t *testing.T, d *Dandelion) {
	RepeatVote(t, d, repCrtName, tester.bpList, tester.acc4.Name)
}

//
// proposal new admin when last proposal has not expired
//
func (tester *ReputationTester) multipleProposals(t *testing.T, d *Dandelion)  {
	MultipleProposals(t, d, repCrtName, tester.bpList)

}

//
// Successfully set up the administrator
//
func (tester *ReputationTester) setAdminSuccess(t *testing.T, d *Dandelion)  {
	a := assert.New(t)
	a.True(d.Contract(constants.COSSysAccount, repCrtName).CheckExist())
	admin := tester.acc4.Name
	ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.proposal %q", tester.acc0.Name, constants.COSSysAccount, repCrtName, admin))
	for i := 0; i < tester.threshold; i++ {
		name := fmt.Sprintf("actor%d", i)
		ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.vote ", name, constants.COSSysAccount, repCrtName))
	}
	d.ProduceBlocks(SysCtrPropExpTime + 1)
	a.NotNil(d.GlobalProps().ReputationAdmin)
	a.Equal(d.GlobalProps().ReputationAdmin.Value, admin)
}

//non admin call contract to modify others reputation
func (tester *ReputationTester) nonAdminCall(t *testing.T, d *Dandelion)  {
	a := assert.New(t)
	a.True(d.Contract(constants.COSSysAccount, repCrtName).CheckExist())
	admin := d.GlobalProps().ReputationAdmin
	a.NotNil(admin)
	caller := tester.acc0.Name
	a.NotEqual(caller, admin)
	rep1 := tester.acc1.GetReputation()
	repMemo1 := tester.acc1.GetReputationMemo()
	newMemo := repMemo1 + "1"
	ApplyError(t, d, fmt.Sprintf("%s: %s.%s.setrep %q, %d, %q", caller, constants.COSSysAccount, repCrtName, tester.acc1.Name, rep1+1, newMemo))
	a.Equal(tester.acc1.GetReputation(), rep1)
	a.Equal(tester.acc1.GetReputationMemo(), repMemo1)
}

//admin call contract which owner is't system account to modify others reputation
func (tester *ReputationTester) callWrongContract(t *testing.T, d *Dandelion)  {
	a := assert.New(t)
	a.True(d.Contract(tester.acc0.Name, repCrtName).CheckExist())
	admin := d.GlobalProps().GetReputationAdmin()
	a.NotNil(admin)
	rep2 := tester.acc2.GetReputation()
	repMemo2 := tester.acc2.GetReputationMemo()
	newMemo := GetNewMemo(repMemo2)
	ApplyError(t, d, fmt.Sprintf("%s: %s.%s.setrep %q, %d, %q", admin.Value, tester.acc0.Name, repCrtName, tester.acc2.Name, rep2+1, newMemo))
    a.Equal(rep2, tester.acc2.GetReputation())
	a.Equal(repMemo2, tester.acc2.GetReputationMemo())
}

//modify not exist account's reputation
func (tester *ReputationTester) mdNotExiAcct(t *testing.T, d *Dandelion)  {
	a := assert.New(t)
	a.True(d.Contract(constants.COSSysAccount, repCrtName).CheckExist())
	admin := d.GlobalProps().ReputationAdmin
	a.NotNil(admin)
	mdAcct := d.Account("account2")
	a.False(mdAcct.CheckExist())
	ApplyError(t, d, fmt.Sprintf("%s: %s.%s.setrep %q, %d, %q", admin.Value, constants.COSSysAccount, repCrtName, mdAcct.Name, 10, "memo"))
	a.False(d.Account(mdAcct.Name).CheckExist())
}

func (tester *ReputationTester) repOverFlow(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.True(d.Contract(constants.COSSysAccount, repCrtName).CheckExist())
	admin := d.GlobalProps().ReputationAdmin
	a.NotNil(admin)
	rep3 := tester.acc3.GetReputation()
	repMemo3 := tester.acc3.GetReputationMemo()
	newMemo := GetNewMemo(repMemo3)
	mdRep := constants.MaxReputation + 100
	ApplyError(t, d, fmt.Sprintf("%s: %s.%s.setrep %q, %d, %q", admin.Value, constants.COSSysAccount, repCrtName, tester.acc3.Name, mdRep, newMemo))
	a.Equal(rep3, tester.acc3.GetReputation())
	a.Equal(repMemo3, tester.acc3.GetReputationMemo())
}

func (tester *ReputationTester) successMdRep(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.True(d.Contract(constants.COSSysAccount, repCrtName).CheckExist())
	admin := d.GlobalProps().ReputationAdmin
	a.NotNil(admin)
	rep4 := tester.acc4.GetReputation()
	repMemo4 := tester.acc3.GetReputationMemo()
	newMemo := GetNewMemo(repMemo4)
	mdRep := uint32(constants.MaxReputation)/2
	if mdRep == rep4 {
		mdRep += 1
	}
	ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.setrep %q, %d, %q", admin.Value, constants.COSSysAccount, repCrtName, tester.acc4.Name, mdRep, newMemo))
	a.Equal(mdRep, tester.acc4.GetReputation())
	a.Equal(newMemo, tester.acc4.GetReputationMemo())
}


//account register as bp After reputation modified to min value
func (tester *ReputationTester) regBpWithMinRep(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	modWithMinReputation(t, d, tester.acc6)

	//bp register
	a.Error(RegisterBp([]*DandelionAccount{tester.acc6}, d))
}

func (tester *ReputationTester) minReputationCashout(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	//bp register
	a.NoError(RegisterBp([]*DandelionAccount{tester.acc7}, d))
	bpWrap := d.BlockProducer(tester.acc7.Name)
	a.True(bpWrap.CheckExist())
	a.True(bpWrap.GetBpVest().Active)
	oldBpVest := bpWrap.GetBpVest().VoteVest.Value

	modWithMinReputation(t, d, tester.acc7)

	a.False(bpWrap.GetBpVest().Active)
	a.Equal(oldBpVest, bpWrap.GetBpVest().VoteVest.Value)

	title := "title7"
	content := "content7"
	pId, err := PostArticle(tester.acc7, title, content, []string{"tag7"}, d)
	if a.NoError(err) {
		// vote to the article
		a.NoError( VoteToPost(tester.acc6, pId) )

		// reply to article
		rId, err := ReplyArticle(tester.acc7, pId, "test reply")
		a.NoError(err)

		// vote to the reply
		a.NoError( VoteToPost(tester.acc6, rId) )
	}

	authorWrap := d.Account(tester.acc7.Name).SoAccountWrap
	oldVest := authorWrap.GetVest()
	a.NoError( d.ProduceBlocks(constants.PostCashOutDelayBlock + 1) )
	newVest := authorWrap.GetVest()
	a.Equal(oldVest.Value, newVest.Value)
}

func modWithMinReputation (t *testing.T, d *Dandelion, account *DandelionAccount) {
	a := assert.New(t)

	a.True(d.Contract(constants.COSSysAccount, repCrtName).CheckExist())
	admin := d.GlobalProps().ReputationAdmin
	a.NotNil(admin)
	mdRep := uint32(constants.MinReputation)
	newMemo := GetNewMemo(account.GetReputationMemo())
	//modify account reputation to MinReputation
	ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.setrep %q, %d, %q", admin.Value, constants.COSSysAccount, repCrtName,account.Name, mdRep, newMemo))
	a.Equal(mdRep, account.GetReputation())
	a.Equal(newMemo, account.GetReputationMemo())
}

//vote when a account's reputation is modified to min value
func (tester *ReputationTester) voteWithMinRep(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.True(d.Contract(constants.COSSysAccount, repCrtName).CheckExist())
	admin := d.GlobalProps().ReputationAdmin
	a.NotNil(admin)
	a.Equal(uint32(constants.MinReputation), tester.acc6.GetReputation())
	mdRep := uint32(constants.MinReputation)
	newMemo := GetNewMemo(tester.acc6.GetReputationMemo())
	//modify to min reputation
	name5 := tester.acc5.Name
	ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.setrep %q, %d, %q", admin.Value, constants.COSSysAccount, repCrtName, name5, mdRep, newMemo))
	a.Equal(mdRep, tester.acc6.GetReputation())
	a.Equal(newMemo, tester.acc5.GetReputationMemo())
	//post article
	title := "title"
	content := "content"
	pId,err := PostArticle(tester.acc0, title, content, []string{"tag"}, d)
	if a.NoError(err) {
		pWrap := d.Post(pId)
		a.True(pWrap.CheckExist())
		oriWeight := pWrap.GetWeightedVp()
		//vote to post, if voter's reputation is MinReputation, he has no voting power
		a.NoError(tester.acc5.SendTrxAndProduceBlock(Vote(name5, pId)))
		a.Equal(oriWeight, pWrap.GetWeightedVp())
	}

}


//vote use ticket when a account's reputation is modified to min value
func (tester *ReputationTester) voteByTicketWithMinRep(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	acctName := "account3"
	a.False(d.Account(acctName).CheckExist())
	priv, _ := prototype.GenerateNewKey()
	pub, _ := priv.PubKey()
	var (
		ops []*prototype.Operation
		transAmount uint64 = 10000 * constants.COSTokenDecimals
		ticketCnt uint64 = 2
	)

	ops = append(ops,
		AccountCreate(constants.COSInitMiner, acctName, pub, constants.DefaultAccountCreateFee, ""),
		Transfer(constants.COSInitMiner, acctName, transAmount, ""))
	a.NoError(d.SendTrxByAccount(constants.COSInitMiner, ops...))
	a.NoError(d.ProduceBlocks(1))
	acct := d.Account(acctName)
	a.True(acct.CheckExist())
	d.PutAccount(acctName, priv)
    //Get ticket
    a.NoError(acct.SendTrxAndProduceBlock(AcquireTicket(acctName, ticketCnt)))
	a.Equal(acct.GetChargedTicket(), uint32(ticketCnt))
    //modify reputation to MinReputation
	a.True(d.Contract(constants.COSSysAccount, repCrtName).CheckExist())
	admin := d.GlobalProps().ReputationAdmin
	a.NotNil(admin)
    var minRep uint32 = constants.MinReputation
	newMemo := "newMemo"
	ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.setrep %q, %d, %q", admin.Value, constants.COSSysAccount, repCrtName, acctName, minRep, newMemo))
	a.Equal(minRep, acct.GetReputation())
	a.Equal(newMemo, acct.GetReputationMemo())

	title := "title1"
	content := "content1"
	pId,err := PostArticle(tester.acc1, title, content, []string{"tag1"}, d)
	if a.NoError(err) {
		pWrap := d.Post(pId)
		a.True(pWrap.CheckExist())
		oriTicket := pWrap.GetTicket()
		//use ticket to vote to post, if voter's reputation is MinReputation, his tickets are useless
		a.NoError(acct.SendTrxAndProduceBlock(VoteByTicket(acctName, pId, ticketCnt)))
		a.Equal(oriTicket, pWrap.GetTicket())
	}
}

func (tester *ReputationTester) mdRepMemoToEmptyStr(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.True(d.Contract(constants.COSSysAccount, repCrtName).CheckExist())
	admin := d.GlobalProps().ReputationAdmin
	a.NotNil(admin)
	mdRep := uint32(tester.acc3.GetReputation() + 1)
	memo := "memo"
	//modify actor6's reputation to MinReputation
	ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.setrep %q, %d, %q", admin.Value, constants.COSSysAccount, repCrtName,tester.acc6.Name, mdRep, memo))
	a.Equal(mdRep, tester.acc6.GetReputation())
	a.Equal(memo, tester.acc6.GetReputationMemo())
	//modify memo with ""
	newMemo := ""
	ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.setrep %q, %d, %q", admin.Value, constants.COSSysAccount, repCrtName,tester.acc6.Name, mdRep, newMemo))
	a.Equal(mdRep, tester.acc6.GetReputation())
	a.Equal(newMemo, tester.acc6.GetReputationMemo())
}

func (tester *ReputationTester) updateAdmin(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.True(d.Contract(constants.COSSysAccount, repCrtName).CheckExist())
	admin := d.GlobalProps().ReputationAdmin
	a.NotNil(admin)
	newAdmin := tester.acc5
	a.NotEqual(admin.Value, newAdmin.Name)
	ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.proposal %q", tester.acc0.Name, constants.COSSysAccount, repCrtName, newAdmin.Name))
	for i := 0; i < tester.threshold; i++ {
		name := fmt.Sprintf("actor%d", i)
		ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.vote ", name, constants.COSSysAccount, repCrtName))
	}
	d.ProduceBlocks(SysCtrPropExpTime + 1)
	a.NotNil(d.GlobalProps().ReputationAdmin)
	a.Equal(d.GlobalProps().ReputationAdmin.Value, newAdmin.Name)

	//new admin work
	rep6 := tester.acc6.GetReputation()
	repMemo6 := tester.acc6.GetReputationMemo()
	newRep6 := rep6 + 1
	if newRep6 > constants.MaxReputation {
		newRep6 = 1
	}
	newRepMemo6 := "test" + repMemo6
	ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.setrep %q, %d, %q", newAdmin.Name, constants.COSSysAccount, repCrtName,tester.acc6.Name, newRep6, newRepMemo6))
    a.Equal(newRep6, tester.acc6.GetReputation())
	a.Equal(newRepMemo6 , tester.acc6.GetReputationMemo())

	//origin admin not work
	rep2 := tester.acc2.GetReputation()
	repMemo2 := tester.acc2.GetReputationMemo()
	newRep2 := rep2 + 1
	if newRep2 > constants.MaxReputation {
		newRep2 = 1
	}
	ApplyError(t, d, fmt.Sprintf("%s: %s.%s.setrep %q, %d, %q", admin.Value, constants.COSSysAccount, repCrtName,tester.acc2.Name, newRep2, repMemo2))
    a.Equal(rep2, tester.acc2.GetReputation())
    a.NotEqual(newRep2, tester.acc2.GetReputation())
}