package vm

import (
	"fmt"
	"testing"
)

func TestCosVM_Byte(t *testing.T) {
	a := []byte{'a', 'b', 0}
	b := []byte("ab")
	b = append(b, 0)
	for _, i := range a {
		if i == 0 {
			fmt.Println("hello a")
		}
	}
	for _, i := range b {
		if i == 0 {
			fmt.Println("hello b")
		}
	}
}

func addByte(buf *[]byte) {
	*buf = append(*buf, 'a')
}

func TestCosVM_Byte2(t *testing.T) {
	var buf []byte
	addByte(&buf)
	fmt.Println(buf)
}
