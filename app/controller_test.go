package app

import (
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/prototype"
	"testing"
)

const (
	accountName = "bob"
	pubKey = "COS6oLVaFEtHZmPDuCvuB48NpSKytjyavPk5MwtN4HqKG16oSA2wS"
	priKey = "EpgwWxboEdaWfEBdWswobsBt8pBF6xoYQPayBs4eVysMGGGYL"

)


func createSigTrx(op interface{}) (*prototype.SignedTransaction,error) {

	privKey, err := prototype.PrivateKeyFromWIF(constants.INITMINER_PRIKEY)
	if err != nil {
		return nil, err
	}

	tx := &prototype.Transaction{RefBlockNum: 0, RefBlockPrefix: 0, Expiration: &prototype.TimePointSec{UtcSeconds: 0}}
	tx.AddOperation(op)

	signTx := prototype.SignedTransaction{Trx: tx}

	res := signTx.Sign(privKey, prototype.ChainId{Value: 0})
	signTx.Signatures = append(signTx.Signatures, &prototype.SignatureType{Sig: res})

	return &signTx, nil
}

func Test_PushTrx(t *testing.T) {
	clearDB()

	pub,err := prototype.PublicKeyFromWIF(pubKey)
	if err != nil {
		t.Error("PublicKeyFromWIF error")
	}
	acop := &prototype.AccountCreateOperation{
		Fee:            prototype.MakeCoin(1),
		Creator:        &prototype.AccountName{Value: constants.COS_INIT_MINER},
		NewAccountName: &prototype.AccountName{Value: accountName},
		Owner: &prototype.Authority{
			Cf:              prototype.Authority_owner,
			WeightThreshold: 1,
			AccountAuths: []*prototype.KvAccountAuth{
				&prototype.KvAccountAuth{
					Name:    &prototype.AccountName{Value: constants.COS_INIT_MINER},
					Weight: 3,
				},
			},
			KeyAuths: []*prototype.KvKeyAuth{
				&prototype.KvKeyAuth{
					Key: pub,	// owner key
					Weight: 1,
				},
			},
		},
		Active: &prototype.Authority{
		},
		Posting: &prototype.Authority{
		},
		MemoKey: pub, // new account's memo key
	}

	signedTrx, err := createSigTrx(acop)
	if err != nil {
		t.Error("createSigTrx failed:",err)
	}

	// set up controller
	db := startDB()
	defer db.Close()
	c := startController(db)

	invoice := c.PushTrx(signedTrx)
	if invoice.Status != 200 {
		t.Error("PushTrx return status error:",invoice.Status)
	}

	bobName := &prototype.AccountName{Value:accountName}
	bobWrap := table.NewSoAccountWrap(db,bobName)
	if !bobWrap.CheckExist() {
		t.Error("create account failed")
	}
}