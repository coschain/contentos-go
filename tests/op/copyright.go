package op

import (
	"fmt"
	"github.com/coschain/contentos-go/cmd/wallet-cli/commands/utils"
	"github.com/coschain/contentos-go/common/constants"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/stretchr/testify/assert"
	"testing"
)

const crCrtName = "copyright"

type CopyrightTester struct {
	acc0, acc1, acc2, acc3, acc4, acc5, acc6  *DandelionAccount
	bpNum, threshold int
	bpList []*DandelionAccount
}

func (tester* CopyrightTester) Test(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")
	tester.acc3 = d.Account("actor3")
	tester.acc4 = d.Account("actor4")
	tester.acc5 = d.Account("actor5")
	tester.acc6 = d.Account("actor6")

	a.NoError(DeploySystemContract(constants.COSSysAccount, crCrtName, d))
	a.True(d.Contract(constants.COSSysAccount, crCrtName).CheckExist())
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
	t.Run("modify non exist post", d.Test(tester.mdNotExiPost))
	t.Run("modify wrong status", d.Test(tester.mdWrongStatus))
	t.Run("success modify post", d.Test(tester.mdSuccess))
	t.Run("reply infringement", d.Test(tester.replyInfringement))
	t.Run("modify empty memo", d.Test(tester.mdEmptyMemo))
	t.Run("modify admin", d.Test(tester.mdAdmin))
}

func (tester* CopyrightTester) nonBpProposalAdmin(t *testing.T, d *Dandelion) {
	NonBpProposalAdmin(t, d, tester.acc4.Name, tester.acc0.Name, crCrtName)
}


func (tester* CopyrightTester) proposalWrongBp(t *testing.T, d *Dandelion) {
	ProposalWrongBp(t, d, tester.acc0.Name, crCrtName)
}

func (tester* CopyrightTester) insufficientBpVote(t *testing.T, d *Dandelion) {
	InsufficientBpVote(t, d, crCrtName, tester.acc4.Name, tester.bpList)
}

//
// total vote count more than 2/3,but some voters is't bp
//
func (tester* CopyrightTester) nonBpVote(t *testing.T, d *Dandelion) {
	NonBpVote(t, d, crCrtName, tester.acc4.Name,tester.bpList, []*DandelionAccount{tester.acc4})
}

//
// the bp vote count greater than 2/3 but the proposal has expired
//
func (tester* CopyrightTester) reachThresholdWhenExpire(t *testing.T, d *Dandelion) {
	ReachThresholdWhenExpire(t, d, crCrtName, tester.acc4.Name,tester.bpList)
}

func (tester* CopyrightTester) repeatVote(t *testing.T, d *Dandelion) {
	RepeatVote(t, d, crCrtName, tester.bpList, tester.acc4.Name)
}

//
// proposal new admin when last proposal has not expired
//
func (tester* CopyrightTester) multipleProposals(t *testing.T, d *Dandelion) {
	MultipleProposals(t, d, crCrtName, tester.bpList)
}


func (tester *CopyrightTester) setAdminSuccess(t *testing.T, d *Dandelion)  {
	a := assert.New(t)
	a.True(d.Contract(constants.COSSysAccount, crCrtName).CheckExist())
	admin := tester.acc4.Name
	a.Nil(d.GlobalProps().GetCopyrightAdmin())
	ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.proposal %q", tester.acc0.Name, constants.COSSysAccount, crCrtName, admin))
	for i := 0; i < tester.threshold; i++ {
		name := fmt.Sprintf("actor%d", i)
		ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.vote ", name, constants.COSSysAccount, crCrtName))
	}
	d.ProduceBlocks(SysCtrPropExpTime + 1)
	a.NotNil(d.GlobalProps().CopyrightAdmin)
	a.NotEqual(len(d.GlobalProps().CopyrightAdmin.Value), 0)
	a.Equal(d.GlobalProps().CopyrightAdmin.Value, admin)
}


func (tester *CopyrightTester) nonAdminCall(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	//post article
	title := "title"
	content := "content"
	pId,err := PostArticle(tester.acc0, title, content, []string{"tag"}, d)
	if a.NoError(err) {
		pWrap := d.Post(pId)
		a.True(pWrap.CheckExist())
		a.True(d.Contract(constants.COSSysAccount, crCrtName).CheckExist())
		admin := d.GlobalProps().CopyrightAdmin
		a.NotNil(admin)
		caller := tester.acc0.Name
		a.NotEqual(caller, admin)
		cr := pWrap.GetCopyright()
		newCr := tester.getNewCopyrightStatus(cr)
        a.NotEqual(newCr, cr)
		crMemo := pWrap.GetCopyrightMemo()
		newMemo := GetNewMemo(crMemo)
		a.NotEqual(newMemo, crMemo)
		ApplyError(t, d, fmt.Sprintf("%s: %s.%s.setcopyright %v,%d,%q", caller, constants.COSSysAccount, crCrtName, pId, newCr, newMemo))
		a.Equal(pWrap.GetCopyright(), cr)
		a.Equal(pWrap.GetCopyrightMemo(), crMemo)
	}
}


func (tester *CopyrightTester) callWrongContract(t *testing.T, d *Dandelion)  {
	a := assert.New(t)
	a.True(d.Contract(tester.acc0.Name, crCrtName).CheckExist())
	admin := d.GlobalProps().CopyrightAdmin
	a.NotNil(admin)
	a.NotEqual(admin.Value, 0)
	title := "title2"
	content := "content2"
	pId,err := PostArticle(tester.acc2, title, content, []string{"tag"}, d)
	if a.NoError(err) {
		pWrap := d.Post(pId)
		a.True(pWrap.CheckExist())
		cr := pWrap.GetCopyright()
		crMemo := pWrap.GetCopyrightMemo()
		newMemo := GetNewMemo(crMemo)
		a.NotEqual(crMemo, newMemo)
		newCr := tester.getNewCopyrightStatus(cr)
		a.NotEqual(newCr, cr)
		ApplyError(t, d, fmt.Sprintf("%s: %s.%s.setcopyright %v,%d,%q", admin.Value, tester.acc0.Name, crCrtName, pId, newCr, newMemo))
		a.Equal(cr, pWrap.GetCopyright())
		a.Equal(crMemo, pWrap.GetCopyrightMemo())
	}
	
	
}

func (tester *CopyrightTester) mdNotExiPost(t *testing.T, d *Dandelion)  {
	a := assert.New(t)
	a.True(d.Contract(constants.COSSysAccount, crCrtName).CheckExist())
	admin := d.GlobalProps().CopyrightAdmin
	a.NotNil(admin)
	postId := utils.GenerateUUID("account2" + "t1")
	pWrap := d.Post(postId)
	a.False(pWrap.CheckExist())
	memo := "memo"
	ApplyError(t, d, fmt.Sprintf("%s: %s.%s.setcopyright %v,%d,%q", admin.Value, constants.COSSysAccount, crCrtName, postId, uint32(constants.CopyrightConfirmation), memo))
    a.False(d.Post(postId).CheckExist())
}

//
// modified copyright status is not CopyrightUnkown:0 CopyrightInfringement:1 CopyrightConfirmation:2
//
func (tester *CopyrightTester) mdWrongStatus(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.True(d.Contract(constants.COSSysAccount, crCrtName).CheckExist())
	admin := d.GlobalProps().CopyrightAdmin
	a.NotNil(admin)
	//post
	title := "title1"
	content := "content1"
	pId,err := PostArticle(tester.acc1, title, content, []string{"tag"}, d)
	if a.NoError(err) {
		pWrap := d.Post(pId)
		a.True(pWrap.CheckExist())
		cr := pWrap.GetCopyright()
		newCr := 4
		crMemo :=pWrap.GetCopyrightMemo()
		newMemo := GetNewMemo(crMemo)
		ApplyError(t, d, fmt.Sprintf("%s: %s.%s.setcopyright %v,%d,%q", admin.Value, constants.COSSysAccount, crCrtName, pId, newCr, newMemo))
		a.Equal(pWrap.GetCopyright(), cr)
		a.Equal(pWrap.GetCopyrightMemo(), crMemo)
	}
}


func (tester *CopyrightTester) mdSuccess(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.True(d.Contract(constants.COSSysAccount, crCrtName).CheckExist())
	admin := d.GlobalProps().CopyrightAdmin
	a.NotNil(admin)
	title := "title4"
	content := "content4"
	pId,err := PostArticle(tester.acc4, title, content, []string{"tag4"}, d)
	if a.NoError(err) {
		pWrap := d.Post(pId)
		a.True(pWrap.CheckExist())
		a.True(d.Contract(constants.COSSysAccount, crCrtName).CheckExist())

		// vote to the article
		a.NoError( VoteToPost(tester.acc3, pId) )

		cr := pWrap.GetCopyright()
		newCr := tester.getNewCopyrightStatus(cr)
		a.NotEqual(newCr, cr)
		crMemo :=pWrap.GetCopyrightMemo()
		newMemo := GetNewMemo(crMemo)
		a.NotEqual(newMemo, crMemo)
		ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.setcopyright %v,%d,%q", admin.Value, constants.COSSysAccount, crCrtName, pId, newCr, newMemo))
		a.Equal(pWrap.GetCopyright(), newCr)
		a.Equal(pWrap.GetCopyrightMemo(), newMemo)
	}

	authorWrap := d.Account(tester.acc4.Name).SoAccountWrap
	oldVest := authorWrap.GetVest()
	a.NoError( d.ProduceBlocks(constants.PostCashOutDelayBlock + 1) )
	newVest := authorWrap.GetVest()
	a.Equal(oldVest.Value, newVest.Value)
}

func (tester *CopyrightTester) replyInfringement(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.True(d.Contract(constants.COSSysAccount, crCrtName).CheckExist())
	admin := d.GlobalProps().CopyrightAdmin
	a.NotNil(admin)
	title := "title6"
	content := "content6"
	pId, err := PostArticle(tester.acc6, title, content, []string{"tag6"}, d)
	a.NoError(err)
	rId, err := ReplyArticle(tester.acc6, pId, "test reply")
	if a.NoError(err) {
		rWrap := d.Post(rId)
		a.True(rWrap.CheckExist())
		a.True(d.Contract(constants.COSSysAccount, crCrtName).CheckExist())

		// vote to the reply
		a.NoError( VoteToPost(tester.acc5, rId) )

		cr := rWrap.GetCopyright()
		newCr := tester.getNewCopyrightStatus(cr)
		a.NotEqual(newCr, cr)
		crMemo :=rWrap.GetCopyrightMemo()
		newMemo := GetNewMemo(crMemo)
		a.NotEqual(newMemo, crMemo)
		ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.setcopyright %v,%d,%q", admin.Value, constants.COSSysAccount, crCrtName, rId, newCr, newMemo))
		a.Equal(rWrap.GetCopyright(), newCr)
		a.Equal(rWrap.GetCopyrightMemo(), newMemo)
	}

	authorWrap := d.Account(tester.acc6.Name).SoAccountWrap
	oldVest := authorWrap.GetVest()
	a.NoError( d.ProduceBlocks(constants.PostCashOutDelayBlock + 1) )
	newVest := authorWrap.GetVest()
	a.Equal(oldVest.Value, newVest.Value)
}

func (tester *CopyrightTester) mdEmptyMemo(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.True(d.Contract(constants.COSSysAccount, crCrtName).CheckExist())
	admin := d.GlobalProps().CopyrightAdmin
	a.NotNil(admin)
	title := "title4"
	content := "content4"
	pId,err := PostArticle(tester.acc4, title, content, []string{"tag4"}, d)
	if a.NoError(err) {
		pWrap := d.Post(pId)
		a.True(pWrap.CheckExist())
		a.True(d.Contract(constants.COSSysAccount, crCrtName).CheckExist())
		cr := pWrap.GetCopyright()
		newCr := tester.getNewCopyrightStatus(cr)

		crMemo :=pWrap.GetCopyrightMemo()
		oldMemo := "memo"
		if len(crMemo) > 0 {
			ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.setcopyright %v,%d,%q", admin, constants.COSSysAccount, crCrtName, pId, newCr, oldMemo))
			a.Equal(oldMemo, pWrap.GetCopyrightMemo())
		}
		newMemo := ""
		ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.setcopyright %v,%d,%q", admin.Value, constants.COSSysAccount, crCrtName, pId, newCr, newMemo))
		a.Equal(pWrap.GetCopyright(), newCr)
		a.Equal(pWrap.GetCopyrightMemo(), newMemo)
	}
}


func (tester *CopyrightTester) mdAdmin(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.True(d.Contract(constants.COSSysAccount, crCrtName).CheckExist())
	oldAdmin := d.GlobalProps().CopyrightAdmin
	a.NotNil(oldAdmin)
	newAdmin := tester.acc5
	a.NotEqual(oldAdmin.Value, newAdmin.Name)
	ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.proposal %q", tester.acc0.Name, constants.COSSysAccount, crCrtName, newAdmin.Name))
	for i := 0; i < tester.threshold; i++ {
		name := fmt.Sprintf("actor%d", i)
		ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.vote", name, constants.COSSysAccount, crCrtName))
	}
	d.ProduceBlocks(SysCtrPropExpTime + 1)
	a.NotNil(d.GlobalProps().CopyrightAdmin)
	a.Equal(d.GlobalProps().CopyrightAdmin.Value, newAdmin.Name)


	title := "title5"
	content := "content5"
	pId,err := PostArticle(tester.acc5, title, content, []string{"tag5"}, d)
	if a.NoError(err) {
		pWrap := d.Post(pId)
		a.True(pWrap.CheckExist())
		//new admin work
		cr := pWrap.GetCopyright()
		newCr := tester.getNewCopyrightStatus(cr)
		a.NotEqual(cr, newCr)
		memo := pWrap.GetCopyrightMemo()
		newMemo := GetNewMemo(memo)
		a.NotEqual(memo, newMemo)
		ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.setcopyright %v,%d,%q", newAdmin.Name, constants.COSSysAccount, crCrtName,  pId, newCr, newMemo))
		a.Equal(newCr, pWrap.GetCopyright())
		a.Equal(newMemo, pWrap.GetCopyrightMemo())
	
		//old admin not work
		mdCr := tester.getNewCopyrightStatus(newCr)
		mdMemo := "test" + newMemo
		ApplyError(t, d, fmt.Sprintf("%s: %s.%s.setcopyright %v,%d,%q", oldAdmin.Value, constants.COSSysAccount, crCrtName, pId, mdCr, mdMemo))
		a.NotEqual(mdCr, pWrap.GetCopyright())
		a.NotEqual(mdMemo, pWrap.GetCopyrightMemo())
		a.Equal(newCr, pWrap.GetCopyright())
		a.Equal(newMemo, pWrap.GetCopyrightMemo())
	
	}

}

func (tester *CopyrightTester) getNewCopyrightStatus(old uint32) uint32 {
	newCr := old
	if newCr < constants.CopyrightConfirmation {
		newCr += 1
	} else {
		newCr = constants.CopyrightUnkown
	}
	return newCr
}