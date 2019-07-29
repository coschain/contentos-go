package contracts

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
	"testing"
)

type NativeTester struct {}

func (tester *NativeTester) Test(t *testing.T, d *Dandelion) {
	t.Run("tests", d.Test(tester.simpleTests))
	t.Run("sha256", d.Test(tester.sha256))
}

func (tester *NativeTester) sha256(t *testing.T, d *Dandelion) {
	data := make([]byte, 16)
	for i := 0; i < 10; i++ {
		_, _ = rand.Reader.Read(data)
		sum := sha256.Sum256(data)
		ApplyNoError(t, d, fmt.Sprintf("actor1: actor1.native_tester.sha256 %v, %v", BytesToJson(data), BytesToJson(sum[:])))
	}
	sum := sha256.Sum256(nil)
	ApplyNoError(t, d, fmt.Sprintf("actor1: actor1.native_tester.sha256 [], %v", BytesToJson(sum[:])))
}

func (tester *NativeTester) blockProducers(d *Dandelion) (names []string) {
	nameList := table.SWitnessOwnerWrap{Dba:d.Database()}
	_ = nameList.ForEachByOrder(nil, nil, nil, nil, func(mVal *prototype.AccountName, sVal *prototype.AccountName, idx uint32) bool {
		if table.NewSoWitnessWrap(d.Database(), mVal).GetActive() {
			names = append(names, mVal.Value)
		}
		return true
	})
	return
}

func (tester *NativeTester) simpleTests(t *testing.T, d *Dandelion) {
	ApplyNoError(t, d, fmt.Sprintf("actor1: actor1.native_tester.current_block_number %d", d.GlobalProps().HeadBlockNumber))
	ApplyNoError(t, d, fmt.Sprintf("actor1: actor1.native_tester.current_timestamp %d", d.GlobalProps().Time.UtcSeconds))
	ApplyNoError(t, d, fmt.Sprintf("actor1: actor1.native_tester.current_witness %q", d.GlobalProps().CurrentWitness.Value))
	ApplyNoError(t, d, fmt.Sprintf("actor1: actor1.native_tester.block_producers %s", StringsToJson(tester.blockProducers(d))))
	ApplyNoError(t, d, "actor1: actor1.native_tester.is_contract_called_by_user true")
	ApplyNoError(t, d, fmt.Sprintf("actor1: actor1.native_tester.get_contract_caller %q", "actor1"))
	ApplyNoError(t, d, fmt.Sprintf("actor0: actor1.native_tester.get_contract_caller %q", "actor0"))
	ApplyNoError(t, d, fmt.Sprintf("actor1: actor1.native_tester.get_contract_caller_contract %q, %q", "", ""))
	ApplyNoError(t, d, fmt.Sprintf("actor1: actor1.native_tester.get_contract_name %q, %q", "actor1", "native_tester"))
	ApplyNoError(t, d, fmt.Sprintf("actor1: actor1.native_tester.get_contract_method %q", "get_contract_method"))

	usrBalance := d.Account("actor0").GetBalance().Value
	contractBalance := d.Contract("actor1", "native_tester").GetBalance().Value
	const amount = 123
	ApplyNoError(t, d, fmt.Sprintf("actor1: actor1.native_tester.get_user_balance %q, %d", "actor0", usrBalance))
	ApplyNoError(t, d, fmt.Sprintf("actor1: actor1.native_tester.get_contract_balance %q, %q, %d", "actor1", "native_tester", contractBalance))
	ApplyNoError(t, d, fmt.Sprintf("actor0: %d actor1.native_tester.get_contract_sender_value %d", amount, amount))
	ApplyNoError(t, d, fmt.Sprintf("actor1: actor1.native_tester.get_user_balance %q, %d", "actor0", usrBalance - amount))
	ApplyNoError(t, d, fmt.Sprintf("actor1: actor1.native_tester.get_contract_balance %q, %q, %d", "actor1", "native_tester", contractBalance + amount))

	ApplyNoError(t, d, fmt.Sprintf("actor1: actor1.native_tester.require_auth %q", "actor1"))
	ApplyError(t, d, fmt.Sprintf("actor1: actor1.native_tester.require_auth %q", "actor0"))
}
