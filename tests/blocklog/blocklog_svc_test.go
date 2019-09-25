//+build !tests

package blocklog

import (
	"fmt"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/iservices"
	"github.com/jinzhu/gorm"
	"testing"
	"time"
)

func TestBlockLogService(t *testing.T) {
	t.Run("block_log_svc",
		NewDandelionTestWithPlugins(true, []string{iservices.BlockLogServiceName, iservices.BlockLogProcessServiceName},
			new(BlockLogServiceTester).Test, sBlockLogTestActors))
}

type BlockLogServiceTester struct {}

type Result struct {
	Balance uint64
}

func (tester *BlockLogServiceTester) Test(t *testing.T, d *Dandelion) {
	new(BlockLogTester).Test(t, d)

	// sleep a while so that block log process service has some time to work
	duration := 5 * time.Second
	fmt.Printf("Sleep for %v...\n", duration)
	time.Sleep(duration)

	db := tester.prepareDatabase(t, d)
	defer func() {
		_ = db.Close()
	}()

	var r Result
	db.Raw("select balance from holders where name=?", "initminer").Scan(&r)
	if r.Balance != 0 {
		t.Fatal("incorrect initminer's balance in database")
	}
}

func (tester *BlockLogServiceTester) prepareDatabase(t *testing.T,d *Dandelion) *gorm.DB {
	config := d.NodeConfig().Database
	connStr := fmt.Sprintf("%s:%s@/%s?charset=utf8mb4&parseTime=True&loc=Local", config.User, config.Password, config.Db)
	db, err := gorm.Open(config.Driver, connStr)
	if err != nil {
		t.Fatal("cannot connect database")
	}
	return db
}