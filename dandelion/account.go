package dandelion

import (
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/prototype"
)

type DandelionAccount struct {
	*table.SoAccountWrap
	D *Dandelion
	Name string
}

func NewDandelionAccount(name string, d *Dandelion) *DandelionAccount {
	return &DandelionAccount{
		SoAccountWrap: table.NewSoAccountWrap(d.Database(), prototype.NewAccountName(name)),
		D: d,
		Name:name,
	}
}

func (acc *DandelionAccount) SendTrx(operations...*prototype.Operation) error {
	return acc.D.SendTrxByAccount(acc.Name, operations...)
}

func (acc *DandelionAccount) SendTrxAndProduceBlock(operations...*prototype.Operation) error {
	receipt, err := acc.D.SendTrxByAccountEx(acc.Name, operations...)

	if err != nil {
		return err
	}
	if !receipt.IsSuccess(){
		return errors.New(fmt.Sprintf("transaction execute fail: %v", receipt.ErrorInfo ) )
	}
	return nil
}


func (acc *DandelionAccount) SendTrxEx(operations...*prototype.Operation) (*prototype.TransactionReceiptWithInfo, error) {
	return acc.D.SendTrxByAccountEx(acc.Name, operations...)
}

func (acc *DandelionAccount) TrxReceipt(operations...*prototype.Operation) *prototype.TransactionReceiptWithInfo {
	return acc.D.TrxReceiptByAccount(acc.Name, operations...)
}
