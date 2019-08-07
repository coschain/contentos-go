package op

import (
	"github.com/coschain/contentos-go/cmd/wallet-cli/commands/utils"
	"github.com/coschain/contentos-go/common/constants"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
	"github.com/stretchr/testify/assert"
	"testing"
)

type ReplyTest struct {
	acc0, acc1, acc2 *DandelionAccount
}

func (tester *ReplyTest) Test(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")

	t.Run("normal", d.Test(tester.normal))
	t.Run("noExistAccountReply", d.Test(tester.noExistAccountReply))
	t.Run("duplicateReplyId", d.Test(tester.duplicateReplyId))
	t.Run("parentIdNoExist", d.Test(tester.parentIdNoExist))
	t.Run("replyDepthOverflow", d.Test(tester.replyDepthOverflow))
}

func (tester *ReplyTest) normal(t *testing.T, d *Dandelion) {
	// post an article
	parentId := doNormalPost(t, d, tester.acc0.Name)

	// repply to a post
	doNormalReply(t, d, tester.acc1.Name, parentId)
}

func (tester *ReplyTest) noExistAccountReply(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	// post an article
	parentId := doNormalPost(t, d, tester.acc0.Name)

	// create no exist account
	noExistAccountName := "invalid"
	createNoExistAccount(noExistAccountName, d)

	// no exist account reply
	replyOp := createReplyOp(noExistAccountName, parentId)
	a.Error( checkError( d.Account(noExistAccountName).TrxReceipt(replyOp) ) )
	replyWrap := d.Post(replyOp.GetOp7().GetUuid())
	a.False(replyWrap.CheckExist())
}

func (tester *ReplyTest) duplicateReplyId(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	// post an article
	rootId := doNormalPost(t, d, tester.acc0.Name)

	// repply to a post
	parentId := doNormalReply(t, d, tester.acc1.Name, rootId)

	// duplicate reply id op
	replyOp1 := createReplyOpWithId(tester.acc1.Name, parentId, rootId)
	a.Error( checkError( d.Account(tester.acc1.Name).TrxReceipt(replyOp1) ) )
}

func (tester *ReplyTest) parentIdNoExist(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	replyOp := createReplyOp(tester.acc1.Name, 123)
	a.Error( checkError( d.Account(tester.acc1.Name).TrxReceipt(replyOp) ) )
	replyWrap := d.Post(replyOp.GetOp7().GetUuid())
	a.False(replyWrap.CheckExist())
}

func (tester *ReplyTest) replyDepthOverflow(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	rootId := doNormalPost(t, d, tester.acc0.Name)

	// continuous reply constants.PostMaxDepth-1 times
	parentId := rootId
	for i:=1;i<constants.PostMaxDepth;i++ {
		tid := doNormalReply(t, d, tester.acc1.Name, parentId)
		parentId = tid
	}

	replyOp := createReplyOp(tester.acc1.Name, parentId)
	a.Error( checkError( d.Account(tester.acc1.Name).TrxReceipt(replyOp) ) )

	// reply article, should no error
	doNormalReply(t, d, tester.acc1.Name, rootId)

	// other account reply, should no error
	doNormalReply(t, d, tester.acc2.Name, rootId)
}

func doNormalReply(t *testing.T, d *Dandelion, name string, parentId uint64) uint64 {
	a := assert.New(t)

	headBlockNumber := d.GlobalProps().HeadBlockNumber
	headBlockTime := d.GlobalProps().Time
	postWrap := d.Post(parentId)
	children := postWrap.GetChildren()
	var rootId uint64
	if postWrap.GetRootId() == 0 {
		rootId = postWrap.GetPostId()
	} else {
		rootId = postWrap.GetRootId()
	}

	replyOp := createReplyOp(name, parentId)
	a.NoError( checkError( d.Account(name).TrxReceipt(replyOp) ) )

	rawOp := replyOp.GetOp7()
	replyWrap := d.Post(rawOp.GetUuid())
	a.True(replyWrap.CheckExist())
	a.Equal(replyWrap.GetPostId(), rawOp.Uuid)
	a.Nil(replyWrap.GetTags())
	a.Equal(replyWrap.GetTitle(), "")
	a.Equal(replyWrap.GetAuthor().Value, rawOp.Owner.Value)
	a.Equal(replyWrap.GetBody(), rawOp.Content)
	a.Equal(replyWrap.GetCreated().UtcSeconds, headBlockTime.UtcSeconds)
	a.Equal(replyWrap.GetCashoutBlockNum(), headBlockNumber + constants.PostCashOutDelayBlock)
	a.Equal(replyWrap.GetDepth(), postWrap.GetDepth()+1)
	a.Equal(replyWrap.GetChildren(), uint32(0))
	a.Equal(replyWrap.GetRootId(), rootId)
	a.Equal(replyWrap.GetParentId(), rawOp.ParentUuid)
	a.Equal(replyWrap.GetWeightedVp(), "0")
	a.Equal(replyWrap.GetVoteCnt(), uint64(0))
	a.Equal(replyWrap.GetBeneficiaries(), rawOp.Beneficiaries)
	a.Equal(replyWrap.GetRewards().Value, uint64(0))
	a.Equal(replyWrap.GetDappRewards().Value, uint64(0))
	a.Equal(replyWrap.GetTicket(), uint32(0))

	authorWrap := d.Account(name).SoAccountWrap
	a.Equal(authorWrap.GetLastPostTime().UtcSeconds, headBlockTime.UtcSeconds)

	a.Equal(postWrap.GetChildren(), children+1)

	return rawOp.GetUuid()
}

func createReplyOp (accName string, parentId uint64) *prototype.Operation {
	postId := utils.GenerateUUID(accName)
	return createReplyOpWithId(accName, postId, parentId)
}

func createReplyOpWithId (accName string, postId, parentId uint64) *prototype.Operation {
	content := "test article for op test"
	beneficiaries := make([]map[string]int, 0)
	return Reply(postId, parentId, accName, content, beneficiaries)
}