package dandelion

import (
	"fmt"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/dandelion/core"
	"github.com/coschain/contentos-go/prototype"
	"github.com/sirupsen/logrus"
	"testing"
)

type Dandelion struct {
	*core.DandelionCore

}

func NewDandelion(logger *logrus.Logger) *Dandelion {
	return &Dandelion{
		DandelionCore: core.NewDandelionCore(logger),
	}
}

type DandelionTestFunc func(*testing.T, *Dandelion)

func NewDandelionTest(f DandelionTestFunc, actors int) func(*testing.T) {
	return func(t *testing.T) {
		d := NewDandelion(nil)
		if d == nil {
			t.Fatal("dandelion creation failed")
		}
		err := d.Start()
		if err != nil {
			t.Fatalf("dandelion start failed: %s", err.Error())
		}
		err = d.CreateAndFund("actor", actors, 1000 * constants.COSTokenDecimals, 10)
		if err != nil {
			t.Fatalf("dandelion createAndFund failed: %s", err.Error())
		}
		defer func() {
			_ = d.Stop()
		}()
		f(t, d)
	}
}

func (d *Dandelion) CreateAndFund(prefix string, n int, coins uint64, fee uint64) error {
	if n <= 0 {
		return nil
	}
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

func (d *Dandelion) Test(f DandelionTestFunc) func(*testing.T) {
	return func(t *testing.T) {
		f(t, d)
	}
}

func (d *Dandelion) GlobalProps() *prototype.DynamicProperties {
	return d.TrxPool().GetProps()
}

func (d *Dandelion) Account(name string) *DandelionAccount {
	return NewDandelionAccount(name, d)
}
