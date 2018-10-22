package common

import (
	"fmt"
	"strings"
	"testing"
)

func TestNAME(t *testing.T) {
	name, err := StringToName("hello")
	if err != nil {
		t.Errorf("invalid StringToName: %s\n", err.Error())
	}
	fmt.Println(name.ToString())
	if strings.Compare(name.ToString(), "hello") != 0 {
		t.Errorf("invalid StringToName: %s\n", "hello")
	}
}
