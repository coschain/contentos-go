package op

import (
	"github.com/coschain/contentos-go/cmd/wallet-cli/commands/utils"
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
}

func doNormalPost(t *testing.T, d *Dandelion, name string) uint64 {
	a := assert.New(t)

	postOp := createPostOp(name)
	a.NoError( checkError( d.Account(name).TrxReceipt(postOp) ) )

	postWrap := d.Post(postOp.GetOp6().GetUuid())
	a.True(postWrap.CheckExist())

	return postOp.GetOp6().GetUuid()
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