package tests

import (
	"encoding/json"
	"fmt"
	"github.com/coschain/contentos-go/app/blocklog"
	"github.com/coschain/contentos-go/common/constants"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBlockLog(t *testing.T) {
	t.Run("block_log", NewDandelionTest(blockLogTest, 10))
}

func blockLogTest(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	_ = d.Node().EvBus.Subscribe(constants.NoticeBlockLog, blockLogRecvd)
	a.NoError(d.Account("actor0").SendTrx(Transfer("actor0", "actor1", 1, "")))
	a.NoError(d.Account("actor2").SendTrx(Transfer("actor2", "actor3", 2, "")))
	a.NoError(d.Account("actor4").SendTrx(Transfer("actor4", "actor5", 3, "")))
	a.NoError(d.Account("actor6").SendTrx(Transfer("actor6", "actor7", 4, "")))
	a.NoError(d.Account("actor8").SendTrx(Transfer("actor8", "actor9", 5, "")))
	a.NoError(d.ProduceBlocks(1))
}

func blockLogRecvd(blockLog *blocklog.BlockLog) {
	j, _ := json.MarshalIndent(blockLog, "", "    ")
	fmt.Println(string(j))
}
