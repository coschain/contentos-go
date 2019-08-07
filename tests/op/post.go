package op

import (
	"github.com/coschain/contentos-go/cmd/wallet-cli/commands/utils"
	"github.com/coschain/contentos-go/common/constants"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
	"github.com/stretchr/testify/assert"
	"testing"
)

type PostTest struct {
	acc0 *DandelionAccount
}

func (tester *PostTest) Test(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")

	t.Run("normal", d.Test(tester.normal))
	t.Run("noExistAccountPost", d.Test(tester.noExistAccountPost))
	t.Run("duplicatePostId", d.Test(tester.duplicatePostId))
}

func (tester *PostTest) normal(t *testing.T, d *Dandelion) {
	doNormalPost(t, d, tester.acc0.Name)
}

func (tester *PostTest) noExistAccountPost(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	accName := "invalid"
	createNoExistAccount(accName, d)

	postOp := createPostOp(accName)
	a.Error( checkError( d.Account(accName).TrxReceipt(postOp) ) )
	postWrap := d.Post(postOp.GetOp6().GetUuid())
	a.False(postWrap.CheckExist())
}

func (tester *PostTest) duplicatePostId(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	postId := doNormalPost(t, d, tester.acc0.Name)

	postOp := createPostOpWithId(tester.acc0.Name, "test post", postId)
	a.Error( checkError( d.Account(tester.acc0.Name).TrxReceipt(postOp) ) )
}

func doNormalPost(t *testing.T, d *Dandelion, name string) uint64 {
	a := assert.New(t)

	headBlockNumber := d.GlobalProps().HeadBlockNumber
	headBlockTime := d.GlobalProps().Time
	totalPostCnt := d.GlobalProps().TotalPostCnt

	postOp := createPostOp(name)
	a.NoError( checkError( d.Account(name).TrxReceipt(postOp) ) )

	rawOp := postOp.GetOp6()
	postWrap := d.Post(rawOp.GetUuid())

	a.True(postWrap.CheckExist())
	a.Equal(postWrap.GetPostId(), rawOp.GetUuid())
	a.Equal(postWrap.GetTags(), rawOp.GetTags())
	a.Equal(postWrap.GetTitle(), rawOp.GetTitle())
	a.Equal(postWrap.GetAuthor().Value, rawOp.GetOwner().Value)
	a.Equal(postWrap.GetBody(), rawOp.GetContent())
	a.Equal(postWrap.GetCreated().UtcSeconds, headBlockTime.UtcSeconds)
	a.Equal(postWrap.GetCashoutBlockNum(), headBlockNumber + constants.PostCashOutDelayBlock)
	a.Equal(postWrap.GetBeneficiaries(), rawOp.GetBeneficiaries())
	a.Equal(postWrap.GetDepth(), uint32(0))
	a.Equal(postWrap.GetChildren(), uint32(0))
	a.Equal(postWrap.GetParentId(), uint64(0))
	a.Equal(postWrap.GetRootId(), uint64(0))
	a.Equal(postWrap.GetWeightedVp(), "0")
	a.Equal(postWrap.GetVoteCnt(), uint64(0))
	a.Equal(postWrap.GetRewards().Value, uint64(0))
	a.Equal(postWrap.GetDappRewards().Value, uint64(0))
	a.Equal(postWrap.GetTicket(), uint32(0))
	a.Equal(postWrap.GetCopyright(), uint32(constants.CopyrightUnkown))

	authorWrap := d.Account(name).SoAccountWrap
	a.Equal(authorWrap.GetLastPostTime().UtcSeconds, headBlockTime.UtcSeconds)

	a.Equal(d.GlobalProps().TotalPostCnt, totalPostCnt+1)

	return rawOp.GetUuid()
}

func createNoExistAccount (accName string, d *Dandelion) {
	priv, _ := prototype.GenerateNewKey()
	d.PutAccount(accName,priv)
}

func createPostOp (accName string) *prototype.Operation {
	title := "test post"
	postId := utils.GenerateUUID(accName + title)
	return createPostOpWithId(accName, title, postId)
}

func createPostOpWithId (accName, title string, postId uint64) *prototype.Operation {
	content := "test article for op test"
	tags := []string{"test"}
	return Post(postId, accName, title, content, tags, nil)
}