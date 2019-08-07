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
	replyOp1 := createReplyOp(tester.acc1.Name, rootId)
	a.NoError( checkError( d.Account(tester.acc1.Name).TrxReceipt(replyOp1) ) )

	replyOp2 := createReplyOp(tester.acc2.Name, rootId)
	a.NoError( checkError( d.Account(tester.acc2.Name).TrxReceipt(replyOp2) ) )
}

func doNormalReply(t *testing.T, d *Dandelion, name string, parentId uint64) uint64 {
	a := assert.New(t)

	replyOp := createReplyOp(name, parentId)
	a.NoError( checkError( d.Account(name).TrxReceipt(replyOp) ) )
	replyWrap := d.Post(replyOp.GetOp7().GetUuid())
	a.True(replyWrap.CheckExist())

	return replyOp.GetOp7().GetUuid()
}

func createReplyOp (accName string, parentId uint64) *prototype.Operation {
	postId := utils.GenerateUUID(accName)
	return createReplyOpWithId(accName, postId, parentId)
}

func createReplyOpWithId (accName string, postId, parentId uint64) *prototype.Operation {
	content := "test article for op test"
	beneficiaries := make(map[string]int)
	return Reply(postId, parentId, accName, content, beneficiaries)
}