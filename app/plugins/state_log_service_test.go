package plugins

import (
	"fmt"
	"testing"
)

func TestBlockLogHeap_Pop(t *testing.T) {
	blockLogHeap := BlockLogHeap{}
	l := blockLogHeap.Len()
	fmt.Println(l)
	i := blockLogHeap.Pop()
	fmt.Println(i)
}
