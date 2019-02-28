package common

import (
	"fmt"
	"testing"
)

func TestInt2Bytes(t *testing.T) {
	r := Int2Bytes(0)
	fmt.Println(r)
}
