package types

import (
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/p2p/common"
	comm "github.com/coschain/contentos-go/p2p/depend/common"
)

// Transaction message
type Trn struct {
	Txn *prototype.SignedTransaction
}

//Serialize message payload
func (this Trn) Serialization(sink *comm.ZeroCopySink) error {
	return this.Txn.Serialization(sink)
}

func (this *Trn) CmdType() string {
	return common.TX_TYPE
}

//Deserialize message payload
func (this *Trn) Deserialization(source *comm.ZeroCopySource) error {
	tx := &prototype.SignedTransaction{}
	err := tx.Deserialization(source)
	if err != nil {
		return err
	}

	this.Txn = tx
	return nil
}
