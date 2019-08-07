package op

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/common/constants"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)


const  (
	frCrtName = "freeze"
    frozen = 0
    unFrozen = 1
    propTabName = "freezetable"
	propIdField = "id"
	maxQueryCnt = 30

)

type proposalId struct {
	Id uint32    `json:"id"`
	Op uint32    `json:"op"`
	Agree uint32   `json:"agree"`
	Accounts  interface{}  `json:"accounts"`
	Memos    string        `json:"memo"`
	Producers interface{} `json:"producers"`
}


type FreezeTester struct {
	acc0, acc1, acc2, acc3, acc4, acc5, acc6  *DandelionAccount
	bpNum, threshold int
	bpList []*DandelionAccount
}

func (tester *FreezeTester) Test(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")
	tester.acc3 = d.Account("actor3")
	tester.acc4 = d.Account("actor4")
	tester.acc5 = d.Account("actor5")
	tester.acc6 = d.Account("actor6")

	a.NoError(DeploySystemContract(constants.COSSysAccount, frCrtName, d))
	a.True(d.Contract(constants.COSSysAccount, frCrtName).CheckExist())
	bpList := []*DandelionAccount{tester.acc0, tester.acc1, tester.acc2, tester.acc3}
	if a.NoError(RegisterBp(bpList, d)) {
		tester.bpNum = len(bpList)
		tester.threshold = tester.bpNum/3*2 + 1
		tester.bpList = append(tester.bpList, bpList...)
	}

	t.Run("caller is not bp", d.Test(tester.nonBpCall))
	t.Run("modify not exist account", d.Test(tester.mdNonExiAcct))
	t.Run("insufficient Bp vote", d.Test(tester.insufficientBpVote))
    t.Run("non Bp Vote", d.Test(tester.nonBpVote))
	t.Run("bp repeat vote", d.Test(tester.repeatVote))
	t.Run("vote wrong proposal id", d.Test(tester.voteWrongProposalId))
	t.Run("modify freeze to wrong status", d.Test(tester.mdWrongStatus))
	t.Run("call wrong contract", d.Test(tester.callWrongContract))
	t.Run("freeze and unfreeze", d.Test(tester.freezeAndUnFreeze))
	t.Run("modify multiple accounts", d.Test(tester.mdMultipleAccounts))
	t.Run("modify multiple non exist accounts", d.Test(tester.mdMulAcctNonExiNon))
	t.Run("modify to empty memo", d.Test(tester.mdEmptyMemo))
}



func (tester *FreezeTester) nonBpCall(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.True(d.Contract(constants.COSSysAccount, frCrtName).CheckExist())
	caller := tester.acc5.Name
	frSta := tester.acc5.GetFreeze()
	memo := tester.acc5.GetFreezeMemo()
	memoArray := StringsToJson([]string{GetNewMemo(memo)})
	nameArray := StringsToJson([]string{tester.acc5.Name})
	ApplyError(t, d, fmt.Sprintf("%s: %s.%s.proposalfreeze %s,%d,%s", caller, constants.COSSysAccount, frCrtName, nameArray, tester.mdFreezeStatus(frSta), memoArray))
	a.Equal(frSta, tester.acc4.GetFreeze())
	a.Equal(memo, tester.acc5.GetFreezeMemo())
}

func (tester *FreezeTester) mdNonExiAcct(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.True(d.Contract(constants.COSSysAccount, frCrtName).CheckExist())
	acctName := "account1"
	a.False(d.Account(acctName).CheckExist())
	memoArray := StringsToJson([]string{"memo1"})
	nameArray := StringsToJson([]string{acctName})
	lastPropId,err := tester.getProposalId(d)
	a.NoError(err)
	ApplyError(t, d, fmt.Sprintf("%s: %s.%s.proposalfreeze %s,%d,%s", tester.acc0.Name, constants.COSSysAccount, frCrtName, nameArray, unFrozen, memoArray))
	newPropId,err := tester.getProposalId(d)
	a.NoError(err)
	a.Equal(lastPropId, newPropId)
}


func (tester *FreezeTester) insufficientBpVote(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.True(d.Contract(constants.COSSysAccount, frCrtName).CheckExist())
	freezeAcct := tester.acc4
	sta := freezeAcct.GetFreeze()
	memo :=  freezeAcct.GetFreezeMemo()
	newSta := tester.mdFreezeStatus(sta)
	a.NotEqual(sta, newSta)
	memoArray := StringsToJson([]string{GetNewMemo(memo)})
	nameArray := StringsToJson([]string{freezeAcct.Name})
	//1.proposal
	ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.proposalfreeze %s,%d,%s", tester.acc0.Name, constants.COSSysAccount, frCrtName, nameArray, newSta, memoArray))
    //2.fetch proposal_id
    propId,err := tester.getProposalId(d)
	a.NoError(err)
    //vote to proposalId
    //only one bp vote, fail to update freeze status
	ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.vote %v", tester.acc0.Name, constants.COSSysAccount, frCrtName, propId))
    a.Equal(sta, freezeAcct.GetFreeze())
    a.Equal(memo, freezeAcct.GetFreezeMemo())

}

//
// total vote count more than 2/3,but some voters is't bp
//
func (tester* FreezeTester) nonBpVote(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.True(d.Contract(constants.COSSysAccount, frCrtName).CheckExist())
	freezeAcct := tester.acc5
	sta := freezeAcct.GetFreeze()
	memo :=  freezeAcct.GetFreezeMemo()
	newSta := tester.mdFreezeStatus(sta)
	a.NotEqual(sta, newSta)
	memoArray,nameArray := tester.getProposalMemoAndNameParams(d,[]*DandelionAccount{freezeAcct})

	//1.proposal
	ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.proposalfreeze %s,%d,%s", tester.acc0.Name, constants.COSSysAccount, frCrtName, nameArray, newSta, memoArray))
	//2.fetch proposal_id
	propId,err := tester.getProposalId(d)
	a.NoError(err)
	//less than 2/3 bp vote to proposalId
	tester.voteById(t, d, propId, 0, tester.threshold-1)
	//non bp vote
	ApplyError(t, d, fmt.Sprintf("%s: %s.%s.vote %v", tester.acc4.Name, constants.COSSysAccount, frCrtName, propId))
	//final vote fail, set freeze fail
	a.Equal(sta, freezeAcct.GetFreeze())
	a.Equal(memo, freezeAcct.GetFreezeMemo())

}

//
// total vote count more than 2/3,but one bp repeat vote
//
func (tester* FreezeTester)  repeatVote(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.True(d.Contract(constants.COSSysAccount, frCrtName).CheckExist())
	freezeAcct := tester.acc6
	sta := freezeAcct.GetFreeze()
	memo :=  freezeAcct.GetFreezeMemo()
	newSta := tester.mdFreezeStatus(sta)
	a.NotEqual(sta, newSta)
	memoArray,nameArray := tester.getProposalMemoAndNameParams(d,[]*DandelionAccount{freezeAcct})

	bp0 := tester.acc0.Name
	//1.proposal
	ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.proposalfreeze %s,%d,%s", bp0, constants.COSSysAccount, frCrtName, nameArray, newSta, memoArray))
	//2.fetch proposal_id
	propId,err := tester.getProposalId(d)
	a.NoError(err)
	//bp0 first vote
	ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.vote %v", bp0, constants.COSSysAccount, frCrtName, propId))
	//2/3 -1 bp vote to proposalId
	tester.voteById(t, d, propId, 1, tester.threshold-1)
	//bp0 repeat vote
	ApplyError(t, d, fmt.Sprintf("%s: %s.%s.vote %v", bp0, constants.COSSysAccount, frCrtName, propId))
	//final vote fail, set freeze fail
	a.Equal(sta, freezeAcct.GetFreeze())
	a.Equal(memo, freezeAcct.GetFreezeMemo())

}


func (tester* FreezeTester) voteWrongProposalId(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.True(d.Contract(constants.COSSysAccount, frCrtName).CheckExist())
	freezeAcct := tester.acc2
	sta := freezeAcct.GetFreeze()
	memo :=  freezeAcct.GetFreezeMemo()
	newSta := tester.mdFreezeStatus(sta)
	a.NotEqual(sta, newSta)
	memoArray,nameArray := tester.getProposalMemoAndNameParams(d,[]*DandelionAccount{freezeAcct})
	ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.proposalfreeze %s,%d,%s", tester.acc0.Name, constants.COSSysAccount, frCrtName, nameArray, newSta, memoArray))
	propId,err := tester.getProposalId(d)
	a.NoError(err)
	tester.voteById(t, d, propId,10, tester.threshold)
	ApplyError(t, d, fmt.Sprintf("%s: %s.%s.vote %v", tester.acc0.Name, constants.COSSysAccount, frCrtName, math.MaxUint32))
	a.Equal(sta, freezeAcct.GetFreeze())
	a.Equal(memo, freezeAcct.GetFreezeMemo())
}

func (tester* FreezeTester) mdWrongStatus(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.True(d.Contract(constants.COSSysAccount, frCrtName).CheckExist())
	freezeAcct := tester.acc3
	sta := freezeAcct.GetFreeze()
	memo :=  freezeAcct.GetFreezeMemo()
	newSta := unFrozen +1
	memoArray,nameArray := tester.getProposalMemoAndNameParams(d,[]*DandelionAccount{freezeAcct})
	lastPropId,err := tester.getProposalId(d)
	a.NoError(err)
	ApplyError(t, d, fmt.Sprintf("%s: %s.%s.proposalfreeze %s,%d,%s", tester.acc0.Name, constants.COSSysAccount, frCrtName, nameArray, newSta, memoArray))
	newPropId,err := tester.getProposalId(d)
	a.NoError(err)
	a.Equal(lastPropId, newPropId)
	a.Equal(sta, freezeAcct.GetFreeze())
	a.Equal(memo, freezeAcct.GetFreezeMemo())

}

func (tester *FreezeTester) callWrongContract(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	bp0 := tester.acc0.Name
	a.True(d.Contract(bp0, frCrtName).CheckExist())
	freezeAcct := tester.acc1
	sta := freezeAcct.GetFreeze()
	newSta := tester.mdFreezeStatus(sta)
	a.NotEqual(sta, newSta)
	lastPropId,err := tester.getProposalId(d)
	a.NoError(err)
	memoArray,nameArray := tester.getProposalMemoAndNameParams(d,[]*DandelionAccount{freezeAcct})
	//call contract which owner is not system account, proposal not work
	ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.proposalfreeze %s,%d,%s", bp0, bp0, frCrtName, nameArray, newSta, memoArray))
	newPropId,err := tester.getProposalId(d)
	a.NoError(err)
	a.Equal(lastPropId, newPropId)

}

func (tester *FreezeTester) freezeAndUnFreeze(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.True(d.Contract(constants.COSSysAccount, frCrtName).CheckExist())

	newAcctName := "account4"

	tester.createNewAcct(t, d, newAcctName)
	newAcct := d.Account(newAcctName)

	freezeAcct := newAcct
	sta := freezeAcct.GetFreeze()
	memo :=  freezeAcct.GetFreezeMemo()
	newSta := tester.mdFreezeStatus(sta)
	a.NotEqual(sta, newSta)
	memoArray,nameArray := tester.getProposalMemoAndNameParams(d,[]*DandelionAccount{freezeAcct})
	newMemo := GetNewMemo(memo)

	//First freeze
	ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.proposalfreeze %s,%d,%s", tester.acc0.Name, constants.COSSysAccount, frCrtName, nameArray, newSta, memoArray))
	propId,err := tester.getProposalId(d)
	a.NoError(err)

	tester.voteById(t, d, propId, 0, tester.threshold)
	a.Equal(newSta, freezeAcct.GetFreeze())
	a.Equal(newMemo, freezeAcct.GetFreezeMemo())

	//unFreeze
	mdSta := tester.mdFreezeStatus(newSta)
	a.NotEqual(mdSta, newSta)
	ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.proposalfreeze %s,%d,%s", tester.acc1.Name, constants.COSSysAccount,
		frCrtName, nameArray, mdSta, memoArray))
	propId,err = tester.getProposalId(d)
	a.NoError(err)

	tester.voteById(t, d, propId, 0, tester.threshold)
	a.Equal(mdSta, freezeAcct.GetFreeze())
}

func (tester *FreezeTester) mdMultipleAccounts(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.True(d.Contract(constants.COSSysAccount, frCrtName).CheckExist())
	newAcctName1 := "account6"
	newAcctName2 := "account7"
	tester.createNewAcct(t, d, newAcctName1)
	tester.createNewAcct(t, d, newAcctName2)
	newAcct1 := d.Account(newAcctName1)
	newAcct2 := d.Account(newAcctName2)

	sta1 := newAcct1.GetFreeze()
	memo1 :=  newAcct1.GetFreezeMemo()
	newSta1 := tester.mdFreezeStatus(sta1)
	a.NotEqual(sta1, newSta1)
    newMemo1 := GetNewMemo(memo1)
	sta2 := newAcct2.GetFreeze()
	memo2 :=  newAcct2.GetFreezeMemo()
	newSta2 := tester.mdFreezeStatus(sta2)
	a.NotEqual(sta2, newSta2)
	newMemo2 := GetNewMemo(memo2)
	acctList := []*DandelionAccount{newAcct1, newAcct2}
	memoArray,nameArray := tester.getProposalMemoAndNameParams(d,acctList)
	//freeze
	mdSta := newSta1
	ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.proposalfreeze %s,%d,%s", tester.acc1.Name, constants.COSSysAccount,
		frCrtName, nameArray, mdSta, memoArray))
	propId,err := tester.getProposalId(d)
	a.NoError(err)
	tester.voteById(t, d, propId, 0, tester.threshold)
	a.Equal(newSta1, newAcct1.GetFreeze())
	a.Equal(newMemo1, newAcct1.GetFreezeMemo())
	a.Equal(newSta2, newAcct2.GetFreeze())
	a.Equal(newMemo2, newAcct2.GetFreezeMemo())

	//unfreeze
	mdSta1 := tester.mdFreezeStatus(newSta1)
	a.NotEqual(mdSta1, newSta1)
	mdSta2 := mdSta1
	a.NotEqual(mdSta2, newSta2)
	ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.proposalfreeze %s,%d,%s", tester.acc1.Name, constants.COSSysAccount,
		frCrtName, nameArray, mdSta1, memoArray))
	propId,err = tester.getProposalId(d)
	a.NoError(err)
	tester.voteById(t, d, propId, 0, tester.threshold)
	a.Equal(mdSta1, newAcct1.GetFreeze())
	a.Equal(newMemo1, newAcct1.GetFreezeMemo())
	a.Equal(mdSta2, newAcct2.GetFreeze())
	a.Equal(newMemo2, newAcct2.GetFreezeMemo())

}

//
// modify multiple accounts at the same time accounts,but some accounts not exist
//
func (tester *FreezeTester) mdMulAcctNonExiNon(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.True(d.Contract(constants.COSSysAccount, frCrtName).CheckExist())
	newAcctName := "account10"
    tester.createNewAcct(t, d, newAcctName)
	newAcct := d.Account(newAcctName)
	acct11 := d.Account("account11")
	acct12 := d.Account("account12")
	a.False(acct11.CheckExist())
	a.False(acct12.CheckExist())
	sta := newAcct.GetFreeze()
	memo := newAcct.GetFreezeMemo()
	newSta := tester.mdFreezeStatus(sta)
	acctList := []*DandelionAccount{newAcct, acct11, acct12}
	memoArray,nameArray := tester.getProposalMemoAndNameParams(d,acctList)
	lastPropId,err := tester.getProposalId(d)
	a.NoError(err)
	ApplyError(t, d, fmt.Sprintf("%s: %s.%s.proposalfreeze %s,%d,%s", tester.acc1.Name, constants.COSSysAccount,
		frCrtName, nameArray, newSta, memoArray))
	a.Equal(sta, newAcct.GetFreeze())
	a.Equal(memo, newAcct.GetFreezeMemo())
	newPropId,err := tester.getProposalId(d)
	a.NoError(err)
	a.Equal(lastPropId, newPropId)

}

func (tester *FreezeTester) mdEmptyMemo(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.True(d.Contract(constants.COSSysAccount, frCrtName).CheckExist())
	newAcctName := "account20"
	tester.createNewAcct(t, d, newAcctName)
	newAcct := d.Account(newAcctName)
	sta := newAcct.GetFreeze()
	memo := newAcct.GetFreezeMemo()
	newMemo := GetNewMemo(memo)
	newSta := tester.mdFreezeStatus(sta)
	memoArray,nameArray := tester.getProposalMemoAndNameParams(d,[]*DandelionAccount{newAcct})
	ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.proposalfreeze %s,%d,%s", tester.acc1.Name, constants.COSSysAccount,
		frCrtName, nameArray, newSta, memoArray))
	propId,err := tester.getProposalId(d)
	a.NoError(err)
	tester.voteById(t, d, propId, 0, tester.threshold)
	a.Equal(newSta, newAcct.GetFreeze())
	a.Equal(newMemo, newAcct.GetFreezeMemo())
	//modify to empty memo
	newMemo = ""
	memoArray = StringsToJson([]string{newMemo})
	ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.proposalfreeze %s,%d,%s", tester.acc1.Name, constants.COSSysAccount,
		frCrtName, nameArray, newSta, memoArray))
	propId,err = tester.getProposalId(d)
	a.NoError(err)
	tester.voteById(t, d, propId, 0, tester.threshold)
	a.Equal(newSta, newAcct.GetFreeze())
	a.Equal(newMemo, newAcct.GetFreezeMemo())
}

func (tester *FreezeTester) getProposalId(d *Dandelion) (uint32, error) {
	var pId uint32 = 0
	tables := d.ContractTables(constants.COSSysAccount, frCrtName)
	if tables == nil {
		return pId, errors.New("no freeze tables exist")
	}
	propTable := tables.Table(propTabName)
	if propTable == nil {
		return pId, errors.New("proposal table not exist")
	}

	res,err := propTable.QueryRecordsJson(propIdField, "", "", true, maxQueryCnt)
	//fmt.Printf("res is %v \n", res)
	var rec []proposalId
	err = json.Unmarshal([]byte(res), &rec)
	if err != nil {
		fmt.Printf("Fail unmarshal  proposal record,error is %v \n", err)
	}
	recNum := len(rec)
	if recNum > 0 {
		pId = rec[0].Id
	}
	return pId,err
}

func (tester *FreezeTester) mdFreezeStatus(old uint32) uint32 {
	newSta := old
	if old == frozen {
		newSta = unFrozen
	} else {
		newSta = frozen
	}
	return newSta
}


func (tester *FreezeTester) getProposalMemoAndNameParams(d *Dandelion, acctList []*DandelionAccount) (string, string) {
	memo := ""
	name := ""
	if len(acctList) > 0 {
		var memoList,nameList []string
		for _,acct := range acctList {
			memoList = append(memoList, GetNewMemo(acct.GetFreezeMemo()))
			nameList = append(nameList, acct.Name)
		}
		if len(nameList) > 0 {
			name = StringsToJson(nameList)
		}

		if len(memoList) > 0 {
			memo = StringsToJson(memoList)
		}
	}
	return  memo,name
}

func (tester *FreezeTester) voteById(t* testing.T,d *Dandelion,id uint32, s int, e int) {
	for i := s; i < e; i++ {
		//bp vote
		name := tester.bpList[i].Name
		ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.vote %v", name, constants.COSSysAccount, frCrtName, id))
	}
}


func (tester *FreezeTester) createNewAcct(t* testing.T,d *Dandelion,name string) {
	a := assert.New(t)
	newAcctName := name
	a.False(d.Account(newAcctName).CheckExist())
	priv, _ := prototype.GenerateNewKey()
	pub, _ := priv.PubKey()
	a.NoError(tester.acc0.SendTrxAndProduceBlock(AccountCreate(tester.acc0.Name, newAcctName, pub, 10, "")))
	newAcct := d.Account(newAcctName)
	a.True(newAcct.CheckExist())
	d.PutAccount(newAcctName, priv)
}