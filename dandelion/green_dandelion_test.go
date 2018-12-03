package dandelion

import (
	"fmt"
	"github.com/coschain/contentos-go/prototype"
	"github.com/inconshreveable/log15"
	"testing"
)

func TestGreenDandelion_GenerateBlock(t *testing.T) {
	log := log15.New()
	dandelion, err := NewDandelion(log)
	if err != nil {
		log.Error("error:", err)
	}
	err = dandelion.OpenDatabase()
	if err != nil {
		log.Error("error:", err)
	}
	acop := &prototype.AccountCreateOperation{
		Fee:            prototype.NewCoin(1),
		Creator:        &prototype.AccountName{Value: "initminer"},
		NewAccountName: &prototype.AccountName{Value: "alice"},
		Owner: &prototype.Authority{
			Cf:              prototype.Authority_owner,
			WeightThreshold: 1,
			AccountAuths: []*prototype.KvAccountAuth{
				&prototype.KvAccountAuth{
					Name:   &prototype.AccountName{Value: "initminer"},
					Weight: 3,
				},
			},
			KeyAuths: []*prototype.KvKeyAuth{
				&prototype.KvKeyAuth{
					Key: &prototype.PublicKeyType{
						Data: []byte{0},
					},
					Weight: 23,
				},
			},
		},
		Active:  &prototype.Authority{},
		Posting: &prototype.Authority{},
	}

	signTx, err := dandelion.Sign(acop)
	if err != nil {
		log.Error("error:", err)
	}
	dandelion.PushTrx(signTx)
	dandelion.GenerateBlock()
	fmt.Println(dandelion.GetProduced())
}
