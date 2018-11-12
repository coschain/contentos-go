package msg

import (
	"fmt"
	"testing"

	"github.com/coschain/contentos-go/common/prototype"
	"github.com/gogo/protobuf/proto"
)

func Test_Serialize(t *testing.T) {
	// transaction
	trx := &prototype.Transaction{
		RefBlockNum:    1,
		RefBlockPrefix: 2,
	}

	sigtrx := new(prototype.SignedTransaction)
	sigtrx.Trx = trx
	msg := new(BroadcastSigTrx)
	msg.SigTrx = sigtrx

	fmt.Printf("before Marshal data:    +%v\n", msg)

	trxdata, err := proto.Marshal(msg)
	if err != nil {
		t.Error("trx Marshal failed")
	}

	var obj BroadcastSigTrx
	err = proto.Unmarshal(trxdata, &obj)
	if err != nil {
		t.Error("trx Marshal failed")
	}

	fmt.Printf("after Unmarshal data:     +%v\n", &obj)
}
