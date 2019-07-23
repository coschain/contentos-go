package utils

import (
	"fmt"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
	"testing"
)

type DandelionTestFunc func(*testing.T, *dandelion.Dandelion)

func NewDandelionTest(f DandelionTestFunc) func(*testing.T) {
	return func(t *testing.T) {
		d := dandelion.NewDandelion(nil)
		if d == nil {
			t.Fatal("dandelion creation failed")
		}
		err := d.Start()
		if err != nil {
			t.Fatalf("dandelion start failed: %s", err.Error())
		}
		err = CreateAndFund(d, "testuser", 10, 10000 * constants.COSTokenDecimals, 10)
		if err != nil {
			t.Fatalf("dandelion createAndFund failed: %s", err.Error())
		}
		defer func() {
			_ = d.Stop()
		}()
		f(t, d)
	}
}

func TestWithDandelion(d *dandelion.Dandelion, f DandelionTestFunc) func(*testing.T) {
	return func(t *testing.T) {
		f(t, d)
	}
}

func CreateAndFund(d *dandelion.Dandelion, prefix string, n int, coins uint64, fee uint64) error {
	var ops []*prototype.Operation
	accounts := make(map[string]*prototype.PrivateKeyType)
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("%s%d", prefix, i)
		priv, _ := prototype.GenerateNewKey()
		pub, _ := priv.PubKey()
		accounts[name] = priv
		ops = append(ops,
			AccountCreate(constants.COSInitMiner, name, pub, fee, ""),
			Transfer(constants.COSInitMiner, name, coins, ""))
	}
	if err := d.SendTrxByAccount(constants.COSInitMiner, ops...); err != nil {
		return err
	} else if err = d.ProduceBlocks(1); err != nil {
		return err
	}
	for name, priv := range accounts {
		d.PutAccount(name, priv)
	}
	return nil
}
