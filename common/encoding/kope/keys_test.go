package kope

import (
	"bytes"
	"testing"
)

func TestKeys(t *testing.T) {
	table := NewKey("accounts")
	pk := AppendKey(table, "pk")
	byBalance := AppendKey(table, "ix_by_balance")
	alice := AppendKey(pk, "alice")
	bob := AppendKey(pk, "bob")
	ikAlice := IndexKey(byBalance, alice, 300.0)
	pkAlice := IndexedPrimaryKey(ikAlice)
	ikBob := IndexKey(byBalance, bob, 2300.0)
	pkBob := IndexedPrimaryKey(ikBob)
	if bytes.Compare(alice, pkAlice) != 0 || bytes.Compare(bob, pkBob) != 0 {
		t.Fatal("IndexedPrimaryKey failed.")
	}
	if bytes.Compare(ikAlice, ikBob) >= 0 {
		t.Fatal("IndexKey not ordered.")
	}
	minPK, maxPK := MinKey(pk), MaxKey(pk)
	if bytes.Compare(alice, minPK) <= 0 || bytes.Compare(alice, maxPK) >= 0 {
		t.Fatal("min max key failed.")
	}
	if bytes.Compare(bob, minPK) <= 0 || bytes.Compare(bob, maxPK) >= 0 {
		t.Fatal("min max key failed.")
	}
}
