package vm

import (
	"fmt"
	"reflect"
	"testing"
)

func A(a int) int {
	return a + 1
}

func TestContext_FuncSig(t *testing.T) {
	funcType := reflect.TypeOf(A)
	if funcType.Kind() == reflect.Func {
		fmt.Println("ahah")
		for i := 0; i < funcType.NumOut(); i++ {
			fmt.Println("param :", i, "type is ", funcType.Out(i))
		}
	}
}
