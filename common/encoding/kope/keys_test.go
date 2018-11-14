package kope

import (
	"bytes"
	"sort"
	"testing"
)

func TestKeys(t *testing.T) {
	table := NewKey("accounts")
	pk := AppendKey(table, "pk")
	byBalance := AppendKey(table, "ix_by_balance")
	byBalanceDesc := AppendKey(table, "ix_by_balance_desc")

	data := map[string]float32{
		"alice": 1000.30,
		"bobo": 500.40,
		"bob": 300.60,
	}
	var primaryKeys Keys
	var byBalanceKeys Keys
	var byBalanceDescKeys Keys
	for name, balance := range data {
		primaryKeys = append(primaryKeys, AppendKey(pk, name))
		byBalanceKeys = append(byBalanceKeys, SingleIndexKey(byBalance, balance, name, false))
		byBalanceDescKeys = append(byBalanceDescKeys, SingleIndexKey(byBalanceDesc, balance, name, true))
	}
	sort.Sort(primaryKeys)
	sort.Sort(byBalanceKeys)
	sort.Sort(byBalanceDescKeys)

	minPK, maxPK := MinKey(pk), MaxKey(pk)
	sortedPK := []string{ "alice", "bob", "bobo" }
	for i := 0; i < len(data); i++ {
		if bytes.Compare(primaryKeys[i], minPK) <= 0 || bytes.Compare(primaryKeys[i], maxPK) >= 0 {
			t.Fatalf("min/max key failed")
		}
		if DecodeKey(primaryKeys[i]).([]interface{})[2] != sortedPK[i] {
			t.Fatalf("primary key not ordered")
		}
	}

	minByBlance, maxByBalance := MinKey(byBalance), MaxKey(byBalance)
	minByBlanceDesc, maxByBalanceDesc := MinKey(byBalanceDesc), MaxKey(byBalanceDesc)
	richPK := []string{ "alice", "bobo", "bob" }
	for i := 0; i < len(data); i++ {
		if bytes.Compare(byBalanceKeys[i], minByBlance) <= 0 || bytes.Compare(byBalanceKeys[i], maxByBalance) >= 0 {
			t.Fatalf("min/max key failed")
		}
		if bytes.Compare(byBalanceDescKeys[i], minByBlanceDesc) <= 0 || bytes.Compare(byBalanceDescKeys[i], maxByBalanceDesc) >= 0 {
			t.Fatalf("min/max key failed")
		}
		if IndexedPrimaryValue(byBalanceDescKeys[i]) != richPK[i] {
			t.Fatalf("balance_desc not ordered")
		}
		if IndexedPrimaryValue(byBalanceKeys[i]) != richPK[len(richPK) - i - 1] {
			t.Fatalf("balance not ordered")
		}
	}
}
