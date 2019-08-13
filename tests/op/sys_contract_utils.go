package op

import (
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/cmd/wallet-cli/commands/utils"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
	"github.com/stretchr/testify/assert"
	"testing"
)

const SysCtrPropExpTime = 5*60

func CreateContractOwner(owner string, d *dandelion.Dandelion) error {
	var err error
	acct := d.Account(owner)
	if !acct.CheckExist() {
		priv, _ := prototype.GenerateNewKey()
		pub, _ := priv.PubKey()
		var ops []*prototype.Operation
		var transAmount uint64 = 1000000 * constants.COSTokenDecimals
		ops = append(ops,
			dandelion.AccountCreate(constants.COSInitMiner, owner, pub, 10, ""),
			dandelion.Transfer(constants.COSInitMiner, owner, transAmount, ""))
		err = d.SendTrxByAccount(constants.COSInitMiner, ops...)
		if err == nil {
			err = d.ProduceBlocks(1)
			if err == nil {
				acct = d.Account(owner)
				if acct.CheckExist() {
					d.PutAccount(owner, priv)
					err = d.SendTrxByAccount(constants.COSInitMiner, dandelion.Stake(constants.COSInitMiner, owner, transAmount))
					if err == nil {
						err = d.ProduceBlocks(1)
					}

				} else {
					err = fmt.Errorf("create account %s failed", owner)
				}
			}
		}
	}
	return err

}

func GetDefChainProps() *prototype.ChainProperties {
	return &prototype.ChainProperties{
		AccountCreationFee: prototype.NewCoin(1),
		StaminaFree:        constants.DefaultStaminaFree,
		TpsExpected:        constants.DefaultTPSExpected,
		EpochDuration:      constants.InitEpochDuration,
		TopNAcquireFreeToken: constants.InitTopN,
		PerTicketPrice:     prototype.NewCoin(1000000),
		PerTicketWeight:    constants.PerTicketWeight,
	}
}

func DeploySystemContract(owner string, contract string, d *dandelion.Dandelion) error {
	err := CreateContractOwner(owner, d)
	if err == nil {
		err = Deploy(d, owner, contract)
	}
	if err != nil {
		fmt.Printf("deploy sys contract error is %v \n", err)
	}
	return err
}

func RegisterBp(list []*dandelion.DandelionAccount, d *dandelion.Dandelion) error {
	for _,acct := range list {
		name := acct.Name
		var ops []*prototype.Operation
		amount := uint64(constants.MinBpRegisterVest*10)
		ops = append(ops, dandelion.TransferToVest(constants.COSInitMiner, name, amount))
		ops = append(ops, dandelion.Stake(constants.COSInitMiner, name, amount))
		err := d.Account(constants.COSInitMiner).SendTrxAndProduceBlock(ops...)
		if err != nil {
			return err
		}
		err = acct.SendTrxAndProduceBlock(dandelion.BpRegister(name, "", "", acct.GetPubKey(), GetDefChainProps()))
		if err != nil {
			return err
		} else {
			witWrap := d.BlockProducer(name)
			if !witWrap.CheckExist() {
				return errors.New(fmt.Sprintf("Fail to register %v to bp", name))
			}
		}
	}
	return  nil

}


func PostArticle(author *dandelion.DandelionAccount, title string, content string,tags []string, d *dandelion.Dandelion) (uint64,error){
	postId := utils.GenerateUUID(author.Name + title)
	rout := map[string]int{author.Name:1}
	beneficiaries :=  []map[string]int{rout}
    err := author.SendTrxAndProduceBlock(dandelion.Post(postId, author.Name, title, content, tags, beneficiaries))
    return postId,err
}

func VoteToPost(author *dandelion.DandelionAccount, postId uint64) error {
	err := author.SendTrxAndProduceBlock(dandelion.Vote(author.Name, postId))
	return err
}

func ReplyArticle(author *dandelion.DandelionAccount, parentId uint64, content string) (uint64, error) {
	postId := utils.GenerateUUID(author.Name)
	rout := map[string]int{author.Name:1}
	beneficiaries :=  []map[string]int{rout}
	err := author.SendTrxAndProduceBlock(dandelion.Reply(postId, parentId, author.Name, content, beneficiaries))
	return postId, err
}

func NonBpProposalAdmin(t *testing.T, d *dandelion.Dandelion, caller string, bp string, contract string)  {
	a := assert.New(t)
	a.True(d.Contract(constants.COSSysAccount, contract).CheckExist())

	witWrap := d.BlockProducer(bp)
	a.True(witWrap.CheckExist())
	witWrap1 := d.BlockProducer(caller)
	a.False(witWrap1.CheckExist())
	//Proposal
	ApplyError(t, d, fmt.Sprintf("%s: %s.%s.proposal %q", caller, constants.COSSysAccount, contract, bp))
	//a.NoError(d.ProduceBlocks(SysCtrPropExpTime + 1))
}


func ProposalWrongBp(t *testing.T, d *dandelion.Dandelion, bp string, contract string) {
	a := assert.New(t)
	a.True(d.Contract(constants.COSSysAccount, contract).CheckExist())
	witWrap := d.BlockProducer(bp)
	a.True(witWrap.CheckExist())
	acct := d.Account("account1")
	a.False(acct.CheckExist())
	ApplyError(t, d, fmt.Sprintf("%s: %s.%s.proposal %q", bp, constants.COSSysAccount, contract, acct.Name))
	//a.NoError(d.ProduceBlocks(SysCtrPropExpTime + 1))

}

func InsufficientBpVote(t *testing.T, d *dandelion.Dandelion, contract string, admin string,
	bpList []*dandelion.DandelionAccount) {
	a := assert.New(t)
	if !a.Nil(d.GlobalProps().ReputationAdmin) {
		a.NotEqual(d.GlobalProps().ReputationAdmin.Value, admin)
	}
	a.NotEmpty(len(bpList))
	a.True(d.Contract(constants.COSSysAccount, contract).CheckExist())
	//proposal actor0
	ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.proposal %q", bpList[0].Name, constants.COSSysAccount, contract, admin))
	//just only one bp vote
	ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.vote", bpList[0].Name, constants.COSSysAccount, contract))
	d.ProduceBlocks(SysCtrPropExpTime + 1)
	//proposal expire,set admin fail
	if !a.Nil(d.GlobalProps().ReputationAdmin) {
		a.NotEqual(d.GlobalProps().ReputationAdmin.Value, admin)
	}
}


func NonBpVote(t *testing.T, d *dandelion.Dandelion, contract string, admin string,
	bpList []*dandelion.DandelionAccount, nonBpList []*dandelion.DandelionAccount) {
	a := assert.New(t)
	if !a.Nil(d.GlobalProps().ReputationAdmin) {
		a.NotEqual(d.GlobalProps().ReputationAdmin.Value, admin)
	}
	bpCnt := len(bpList)
	a.NotEqual(bpCnt, 0)
	nonBpCnt := len(nonBpList)
	threshold := bpCnt/3*2 + 1
	a.NotEqual(nonBpCnt, 0)
	a.True(d.Contract(constants.COSSysAccount, contract).CheckExist())
	ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.proposal %q",bpList[0].Name, constants.COSSysAccount, contract, admin))
	//bp vote count less than 2/3
	for i := 0; i < threshold - 1; i++ {
		name := bpList[i].Name
		ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.vote ", name, constants.COSSysAccount, contract))
	}
	d.ProduceBlocks(1)
	// non bp vote (total vote count reach tester.threshold)
	ApplyError(t, d, fmt.Sprintf("%s: %s.%s.vote", nonBpList[0].Name, constants.COSSysAccount, contract))
	d.ProduceBlocks(SysCtrPropExpTime + 1)
	//total vote count not reach tester.threshold, proposal expire,set admin fail
	if !a.Nil(d.GlobalProps().ReputationAdmin) {
		a.NotEqual(d.GlobalProps().ReputationAdmin.Value, admin)
	}

}


func ReachThresholdWhenExpire(t *testing.T, d *dandelion.Dandelion, contract string, admin string,
	bpList []*dandelion.DandelionAccount) {
	a := assert.New(t)
	if !a.Nil(d.GlobalProps().ReputationAdmin) {
		a.NotEqual(d.GlobalProps().ReputationAdmin.Value, admin)
	}
	a.True(d.Contract(constants.COSSysAccount, contract).CheckExist())
	bpCnt := len(bpList)
	a.NotEqual(bpCnt, 0)
	threshold := bpCnt/3*2 + 1
	ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.proposal %q", bpList[0].Name, constants.COSSysAccount, contract, admin))
	ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.vote", bpList[0].Name, constants.COSSysAccount, contract))
	d.ProduceBlocks(SysCtrPropExpTime + 1)
	for i := 1; i < threshold; i++ {
		name := bpList[i].Name
		ApplyError(t, d, fmt.Sprintf("%s: %s.%s.vote", name, constants.COSSysAccount, contract))
	}
	if !a.Nil(d.GlobalProps().ReputationAdmin) {
		a.NotEqual(d.GlobalProps().ReputationAdmin.Value, admin)
	}
}

func RepeatVote(t *testing.T, d *dandelion.Dandelion, contract string,
	bpList []*dandelion.DandelionAccount, admin string) {
	a := assert.New(t)
	if !a.Nil(d.GlobalProps().ReputationAdmin) {
		a.NotEqual(d.GlobalProps().ReputationAdmin.Value, admin)
	}
	a.True(d.Contract(constants.COSSysAccount, contract).CheckExist())
	bpCnt := len(bpList)
	a.NotEqual(bpCnt, 0)
	threshold := bpCnt/3*2 + 1
	ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.proposal %q", bpList[0].Name, constants.COSSysAccount, contract, admin))
	for i := 0; i < threshold-1; i++ {
		name := bpList[i].Name
		ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.vote", name, constants.COSSysAccount, contract))
	}
	//actor0 repeat vote
	ApplyError(t, d, fmt.Sprintf("%s: %s.%s.vote", bpList[0].Name, constants.COSSysAccount, contract))
	d.ProduceBlocks(SysCtrPropExpTime + 1)
	//cause actor0 repeat vote, so less than 2/3 bp vote, set admin fail
	if !a.Nil(d.GlobalProps().ReputationAdmin) {
		a.NotEqual(d.GlobalProps().ReputationAdmin.Value, admin)
	}
}


func MultipleProposals(t *testing.T, d *dandelion.Dandelion, contract string,
	bpList []*dandelion.DandelionAccount)  {
	a := assert.New(t)
	a.True(d.Contract(constants.COSSysAccount, contract).CheckExist())
	bpCnt := len(bpList)
	a.NotEqual(bpCnt, 0)
	preAdmin := bpList[bpCnt-1].Name
	ApplyNoError(t, d, fmt.Sprintf("%s: %s.%s.proposal %q", bpList[0].Name, constants.COSSysAccount, contract, preAdmin))
	d.ProduceBlocks(1)
	//proposal next admin
	newAdmin := bpList[0].Name
	a.NotEqual(preAdmin, newAdmin)
	ApplyError(t, d, fmt.Sprintf("%s: %s.%s.proposal %q", newAdmin, constants.COSSysAccount, contract, newAdmin))
	d.ProduceBlocks(SysCtrPropExpTime + 1)
	if !a.Nil(d.GlobalProps().ReputationAdmin) {
		a.NotEqual(d.GlobalProps().ReputationAdmin.Value, newAdmin)
	}

}

func GetNewMemo(old string) string {
	return "new" + old
}