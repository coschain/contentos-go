package vme

import (
	"testing"
)

func TestEncodeJsonArray(t *testing.T) {
	jsonStr := `[ 3, 3.4, false, "hello", [2, 4, 5], ["abc", "de"], [[[[[[true]]]]]] ]`
	sig := "d;d;Z;s;[Q;[s;[[[[[[Z"

	_, err := EncodeJsonArrayWithTypeSig(jsonStr, sig)
	if err != nil {
		t.Fatal(err)
	}
}
