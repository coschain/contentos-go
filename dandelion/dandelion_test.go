package dandelion

import (
	"fmt"
	"github.com/coschain/contentos-go/prototype"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDandelion(t *testing.T) {
	a := assert.New(t)
	d := NewDandelion(nil)
	a.NoError(d.Start())

	err := d.SendTrxByAccount("initminer", prototype.GetPbOperation(&prototype.TransferOperation{
		From: prototype.NewAccountName("initminer"),
		To: prototype.NewAccountName("initminer"),
		Amount: prototype.NewCoin(10),
		Memo: "hehe",
	}))
	fmt.Println(err)

	a.NoError(d.ProduceBlocks(1))
	a.NoError(d.Stop())
}
